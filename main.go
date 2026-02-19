package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"

	"github.com/farhanasfar/airbnb-market-scraping-system/config"
	"github.com/farhanasfar/airbnb-market-scraping-system/models"
	"github.com/farhanasfar/airbnb-market-scraping-system/scraper/airbnb"
	"github.com/farhanasfar/airbnb-market-scraping-system/services"
	"github.com/farhanasfar/airbnb-market-scraping-system/storage"
	"github.com/farhanasfar/airbnb-market-scraping-system/utils"
)

func main() {
	// Define CLI flags
	showStats := flag.Bool("show-stats", false, "Show all analytics statistics")
	avgPrice := flag.Bool("avg-price", false, "Show average price")
	maxPrice := flag.Bool("max-price", false, "Show maximum price and property details")
	topRated := flag.Bool("top-rated", false, "Show top 5 highest rated properties")
	byLocation := flag.Bool("by-location", false, "Show listings grouped by location")
	exportCSV := flag.Bool("export-csv", false, "Export listings to CSV file")

	flag.Parse()

	logger := utils.NewLogger()

	// Load configuration
	cfg, err := config.Load("config/config.yaml")
	if err != nil {
		log.Fatal("Failed to load config:", err)
	}

	// Connect to database
	db, err := storage.NewDB(cfg.Database.GetDSN())
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}
	defer db.Close()

	// Create services
	analyticsService := services.NewAnalyticsService(db, logger)
	csvService := services.NewCSVService(db, logger)

	// Handle analytics flags (no scraping needed)
	if *showStats {
		analytics, err := analyticsService.GetAnalytics()
		if err != nil {
			log.Fatal("Failed to get analytics:", err)
		}
		analyticsService.PrintAnalytics(analytics)
		return
	}

	if *avgPrice {
		if err := analyticsService.PrintAveragePrice(); err != nil {
			log.Fatal("Failed to get average price:", err)
		}
		return
	}

	if *maxPrice {
		if err := analyticsService.PrintMaxPrice(); err != nil {
			log.Fatal("Failed to get max price:", err)
		}
		return
	}

	if *topRated {
		if err := analyticsService.PrintTopRated(); err != nil {
			log.Fatal("Failed to get top rated:", err)
		}
		return
	}

	if *byLocation {
		if err := analyticsService.PrintByLocation(); err != nil {
			log.Fatal("Failed to get by location:", err)
		}
		return
	}

	if *exportCSV {
		if err := csvService.ExportToCSV(cfg.Output.CSVFile); err != nil {
			log.Fatal("Failed to export CSV:", err)
		}
		return
	}

	// No flags = run scraping (default behavior)
	runScraping(cfg, db, logger)
}

func runScraping(cfg *config.Config, db *storage.DB, logger *utils.Logger) {
	logger.Info("Starting Airbnb Multi-Location Scraper...")

	// Create services
	listingService := services.NewListingService(db, logger)
	csvService := services.NewCSVService(db, logger)
	analyticsService := services.NewAnalyticsService(db, logger)
	scraper := airbnb.NewScraper(&cfg.Scraper, logger)
	ctx := context.Background()

	// Step 1: Scrape homepage to get location URLs
	logger.Info("\n=== STEP 1: EXTRACTING LOCATIONS FROM HOMEPAGE ===")
	locations, err := scraper.ScrapeHomepageLocations(ctx)
	if err != nil {
		log.Fatal("Failed to scrape homepage:", err)
	}

	if len(locations) == 0 {
		logger.Warning("No locations found on homepage")
		return
	}

	logger.Info("Found %d locations:", len(locations))
	for i, loc := range locations {
		logger.Info("  %d. %s", i+1, loc.Name)
	}

	// Step 2: Scrape properties from each location
	logger.Info("\n=== STEP 2: SCRAPING PROPERTIES FROM EACH LOCATION ===")

	allRawListings := []models.RawListing{}
	totalProperties := 0

	for i, location := range locations {
		logger.Info("\n[%d/%d] Scraping: %s", i+1, len(locations), location.Name)

		// Scrape this location (2 pages × 5 properties = 10 per location)
		rawListings, err := scraper.ScrapeListings(ctx, location.URL)
		if err != nil {
			logger.Error("Failed to scrape %s: %v", location.Name, err)
			continue
		}

		if len(rawListings) == 0 {
			logger.Warning("No listings found for %s", location.Name)
			continue
		}

		logger.Success("Got %d properties from %s", len(rawListings), location.Name)
		allRawListings = append(allRawListings, rawListings...)
		totalProperties += len(rawListings)
	}

	logger.Success("\n=== SCRAPED %d TOTAL PROPERTIES FROM %d LOCATIONS ===",
		totalProperties, len(locations))

	if totalProperties == 0 {
		logger.Warning("No properties scraped, exiting")
		return
	}

	// Print preview if JSON console enabled
	if cfg.Output.JSONConsole {
		logger.Info("\n=== PREVIEW (first 2 listings) ===")
		preview := allRawListings
		if len(preview) > 2 {
			preview = preview[:2]
		}
		jsonData, _ := json.MarshalIndent(preview, "", "  ")
		fmt.Println(string(jsonData))
	}

	// Step 3: Scrape detail pages
	logger.Info("\n=== STEP 3: SCRAPING DETAIL PAGES ===")
	urls := make([]string, 0, len(allRawListings))
	for _, listing := range allRawListings {
		if listing.URL != "" {
			normalizedURL := utils.NormalizeURL(listing.URL)
			urls = append(urls, normalizedURL)
		}
	}

	logger.Info("Scraping details for %d properties...", len(urls))
	detailResults := scraper.ScrapeDetailsWithWorkers(ctx, urls)

	// Merge detail data
	for i := range allRawListings {
		normalizedURL := utils.NormalizeURL(allRawListings[i].URL)
		if detail, ok := detailResults[normalizedURL]; ok && detail.Error == nil {
			allRawListings[i].Bedrooms = detail.Bedrooms
			allRawListings[i].Bathrooms = detail.Bathrooms
			allRawListings[i].Guests = detail.Guests
		}
	}

	// Step 4: Save to database
	logger.Info("\n=== STEP 4: SAVING TO DATABASE ===")
	savedCount, err := listingService.NormalizeAndSave(allRawListings)
	if err != nil {
		logger.Error("Failed to save listings: %v", err)
	}

	// Step 5: Export to CSV
	logger.Info("\n=== STEP 5: EXPORTING TO CSV ===")
	if err := csvService.ExportToCSV(cfg.Output.CSVFile); err != nil {
		logger.Error("Failed to export CSV: %v", err)
	}

	// Step 6: Show analytics
	logger.Info("\n=== STEP 6: ANALYTICS SUMMARY ===")
	analytics, err := analyticsService.GetAnalytics()
	if err != nil {
		logger.Error("Failed to calculate analytics: %v", err)
	} else {
		analyticsService.PrintAnalytics(analytics)
	}

	// Final summary
	logger.Success("\n=== SCRAPING COMPLETE ===")
	logger.Info("Locations scraped: %d", len(locations))
	logger.Info("Total properties found: %d", totalProperties)
	logger.Info("Successfully saved: %d", savedCount)
	logger.Info("CSV file: %s", cfg.Output.CSVFile)
	logger.Info("\n💡 Tip: Run with --show-stats to see analytics anytime!")
	logger.Info("   Other flags: --avg-price, --max-price, --top-rated, --by-location, --export-csv")
}
