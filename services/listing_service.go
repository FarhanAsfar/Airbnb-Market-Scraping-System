package services

import (
	"fmt"

	"github.com/farhanasfar/airbnb-market-scraping-system/models"
	"github.com/farhanasfar/airbnb-market-scraping-system/storage"
	"github.com/farhanasfar/airbnb-market-scraping-system/utils"
)

// ListingService handles business logic for listings
type ListingService struct {
	db     *storage.DB
	logger *utils.Logger
}

// NewListingService creates a new listing service
func NewListingService(db *storage.DB, logger *utils.Logger) *ListingService {
	return &ListingService{
		db:     db,
		logger: logger,
	}
}

// NormalizeAndSave converts raw listings to normalized listings and saves to database
func (scrape *ListingService) NormalizeAndSave(rawListings []models.RawListing) (int, error) {
	if len(rawListings) == 0 {
		return 0, fmt.Errorf("no listings to save")
	}

	successCount := 0
	duplicateCount := 0

	for _, raw := range rawListings {
		// Normalize the data
		listing := scrape.normalize(raw)

		// Skip invalid listings
		if listing.Title == "" || listing.URL == "" {
			scrape.logger.Warning("Skipping invalid listing (no title or URL)")
			continue
		}

		// Save to database (ON CONFLICT handles duplicates)
		err := scrape.db.InsertListing(&listing)
		if err != nil {
			// Log error but continue with other listings
			scrape.logger.Error("Failed to insert listing '%s': %v", listing.Title, err)
			continue
		}

		successCount++
		scrape.logger.Info("✓ Saved: %s ($%.2f)", listing.Title, listing.Price)
	}

	scrape.logger.Success("Saved %d listings to database", successCount)

	if duplicateCount > 0 {
		scrape.logger.Info("Skipped %d duplicate listings", duplicateCount)
	}

	return successCount, nil
}

// normalize converts RawListing to normalized Listing
func (s *ListingService) normalize(raw models.RawListing) models.Listing {
	return models.Listing{
		Title:     raw.Title,
		Price:     utils.NormalizePrice(raw.Price), // "$120" -> 120.0
		Location:  raw.Location,
		Rating:    utils.NormalizeRating(raw.Rating), // "4.95 (123)" -> 4.95
		URL:       raw.URL,
		Bedrooms:  raw.Bedrooms,
		Bathrooms: raw.Bathrooms,
		Guests:    raw.Guests,
	}
}
