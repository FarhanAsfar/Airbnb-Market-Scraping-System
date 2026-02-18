package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/farhanasfar/airbnb-market-scraping-system/config"
	"github.com/farhanasfar/airbnb-market-scraping-system/scraper/airbnb"
	"github.com/farhanasfar/airbnb-market-scraping-system/services"
	"github.com/farhanasfar/airbnb-market-scraping-system/storage"
	"github.com/farhanasfar/airbnb-market-scraping-system/utils"
)

func main() {
	// Initialize logger
	logger := utils.NewLogger()
	logger.Info("Starting Airbnb Scraper...")

	// Load configuration
	cfg, err := config.Load("config/config.yaml")
	if err != nil {
		log.Fatal("Failed to load config:", err)
	}
	logger.Success("Configuration loaded")

	// Connect to database
	db, err := storage.NewDB(cfg.Database.GetDSN())
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}
	defer db.Close()

	// Create services
	listingService := services.NewListingService(db, logger)

	// Create scraper instance
	scraper := airbnb.NewScraper(&cfg.Scraper, logger)

	// Scrape listings
	logger.Info("Starting scraping process...")
	ctx := context.Background()
	rawListings, err := scraper.ScrapeListings(ctx)
	if err != nil {
		log.Fatal("Scraping failed:", err)
	}

	if len(rawListings) == 0 {
		logger.Warning("No listings found. This might mean:")
		logger.Warning("  1. Airbnb changed their HTML structure")
		logger.Warning("  2. The page didn't load properly")
		logger.Warning("  3. Bot detection blocked the request")
		logger.Info("Try setting headless=false in config to see what's happening")
		return
	}

	logger.Success("Scraping completed! Found %d listings", len(rawListings))

	// Print raw listings as JSON
	if cfg.Output.JSONConsole {
		logger.Info("\n=== RAW LISTINGS (JSON) ===")
		jsonData, _ := json.MarshalIndent(rawListings, "", "  ")
		fmt.Println(string(jsonData))
	}

	// Normalize and save to database
	logger.Info("\n=== SAVING TO DATABASE ===")
	savedCount, err := listingService.NormalizeAndSave(rawListings)
	if err != nil {
		logger.Error("Failed to save listings: %v", err)
	}

	logger.Success("\n=== SCRAPING COMPLETE ===")
	logger.Info("Total listings found: %d", len(rawListings))
	logger.Info("Successfully saved: %d", savedCount)
	logger.Info("Duplicates/errors: %d", len(rawListings)-savedCount)
}
