package main

import (
	"context"
	"log"

	"github.com/farhanasfar/airbnb-market-scraping-system/config"
	"github.com/farhanasfar/airbnb-market-scraping-system/scraper/airbnb"
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

	// Create scraper instance
	scraper := airbnb.NewScraper(&cfg.Scraper, logger)

	// Scrape listings
	ctx := context.Background()
	rawListings, err := scraper.ScrapeListings(ctx)
	if err != nil {
		log.Fatal("Scraping failed:", err)
	}

	logger.Success("Scraping completed! Found %d listings", len(rawListings))

}
