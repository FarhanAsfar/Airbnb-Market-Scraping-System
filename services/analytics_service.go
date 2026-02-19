package services

import (
	"fmt"
	"strings"

	"github.com/farhanasfar/airbnb-market-scraping-system/models"
	"github.com/farhanasfar/airbnb-market-scraping-system/storage"
	"github.com/farhanasfar/airbnb-market-scraping-system/utils"
)

// AnalyticsService handles analytics and insights
type AnalyticsService struct {
	db     *storage.DB
	logger *utils.Logger
}

// Analytics holds all calculated statistics
type Analytics struct {
	TotalListings       int
	AveragePrice        float64
	MaxPrice            float64
	MinPrice            float64
	MostExpensive       *models.Listing
	ListingsPerLocation map[string]int
	TopRated            []models.Listing
}

// NewAnalyticsService creates a new analytics service
func NewAnalyticsService(db *storage.DB, logger *utils.Logger) *AnalyticsService {
	return &AnalyticsService{
		db:     db,
		logger: logger,
	}
}

// GetAnalytics calculates all analytics from database
func (s *AnalyticsService) GetAnalytics() (*Analytics, error) {
	listings, err := s.db.GetAllListings()
	if err != nil {
		return nil, fmt.Errorf("failed to get listings: %w", err)
	}

	if len(listings) == 0 {
		return &Analytics{}, nil
	}

	analytics := &Analytics{
		TotalListings:       len(listings),
		ListingsPerLocation: make(map[string]int),
	}

	// Calculate price statistics
	var totalPrice float64
	analytics.MaxPrice = listings[0].Price
	analytics.MinPrice = listings[0].Price
	analytics.MostExpensive = &listings[0]

	for i := range listings {
		listing := &listings[i]

		// Price calculations
		totalPrice += listing.Price

		if listing.Price > analytics.MaxPrice {
			analytics.MaxPrice = listing.Price
			analytics.MostExpensive = listing
		}

		if listing.Price < analytics.MinPrice {
			analytics.MinPrice = listing.Price
		}

		// Location grouping
		analytics.ListingsPerLocation[listing.Location]++
	}

	analytics.AveragePrice = totalPrice / float64(len(listings))

	// Get top 5 rated properties
	analytics.TopRated = s.getTopRated(listings, 5)

	return analytics, nil
}

// getTopRated returns top N highest rated listings
func (s *AnalyticsService) getTopRated(listings []models.Listing, n int) []models.Listing {
	// Sort by rating (descending)
	sorted := make([]models.Listing, len(listings))
	copy(sorted, listings)

	// Simple bubble sort (fine for small datasets)
	for i := 0; i < len(sorted)-1; i++ {
		for j := 0; j < len(sorted)-i-1; j++ {
			if sorted[j].Rating < sorted[j+1].Rating {
				sorted[j], sorted[j+1] = sorted[j+1], sorted[j]
			}
		}
	}

	// Return top N
	if len(sorted) < n {
		return sorted
	}
	return sorted[:n]
}

// PrintAnalytics prints all analytics to console
func (s *AnalyticsService) PrintAnalytics(analytics *Analytics) {
	// Header
	s.logger.Info("\n" + strings.Repeat("=", 70))
	s.logger.Info("              AIRBNB SCRAPING ANALYTICS REPORT")
	s.logger.Info(strings.Repeat("=", 70) + "\n")

	// Total listings
	s.logger.Info("TOTAL LISTINGS: %d\n", analytics.TotalListings)

	// Price statistics
	s.logger.Info("💰 PRICE STATISTICS:")
	s.logger.Info("   Average Price:        $%.2f", analytics.AveragePrice)
	s.logger.Info("   Maximum Price:        $%.2f", analytics.MaxPrice)
	s.logger.Info("   Minimum Price:        $%.2f\n", analytics.MinPrice)

	// Most expensive property
	if analytics.MostExpensive != nil {
		s.logger.Info(" MOST EXPENSIVE PROPERTY:")
		s.logger.Info("   Title:                %s", analytics.MostExpensive.Title)
		s.logger.Info("   Price:                $%.2f per night", analytics.MostExpensive.Price)
		s.logger.Info("   Location:             %s", analytics.MostExpensive.Location)
		s.logger.Info("   Rating:               %.2f ⭐", analytics.MostExpensive.Rating)
		s.logger.Info("   Bedrooms: %d | Bathrooms: %d | Guests: %d\n",
			analytics.MostExpensive.Bedrooms,
			analytics.MostExpensive.Bathrooms,
			analytics.MostExpensive.Guests)
	}

	// Listings per location
	s.logger.Info(" LISTINGS PER LOCATION:")
	for location, count := range analytics.ListingsPerLocation {
		s.logger.Info("   %-35s %d properties", location, count)
	}
	s.logger.Info("")

	// Top rated
	s.logger.Info("⭐ TOP 5 HIGHEST RATED PROPERTIES:")
	for i, listing := range analytics.TopRated {
		s.logger.Info("\n   %d. %s", i+1, listing.Title)
		s.logger.Info("      Rating: %.2f ⭐ | Price: $%.2f | Location: %s",
			listing.Rating, listing.Price, listing.Location)
	}

	// Footer
	s.logger.Info("\n" + strings.Repeat("=", 70) + "\n")
}

// PrintAveragePrice prints only average price
func (s *AnalyticsService) PrintAveragePrice() error {
	analytics, err := s.GetAnalytics()
	if err != nil {
		return err
	}
	s.logger.Info("\nAverage Price: $%.2f\n", analytics.AveragePrice)
	return nil
}

// PrintMaxPrice prints maximum price and property details
func (s *AnalyticsService) PrintMaxPrice() error {
	analytics, err := s.GetAnalytics()
	if err != nil {
		return err
	}

	s.logger.Info("\n MOST EXPENSIVE PROPERTY:")
	if analytics.MostExpensive != nil {
		s.logger.Info("   Title:      %s", analytics.MostExpensive.Title)
		s.logger.Info("   Price:      $%.2f per night", analytics.MostExpensive.Price)
		s.logger.Info("   Location:   %s", analytics.MostExpensive.Location)
		s.logger.Info("   Rating:     %.2f ⭐", analytics.MostExpensive.Rating)
		s.logger.Info("   URL:        %s\n", analytics.MostExpensive.URL)
	}
	return nil
}

// PrintTopRated prints top 5 rated properties
func (s *AnalyticsService) PrintTopRated() error {
	analytics, err := s.GetAnalytics()
	if err != nil {
		return err
	}

	s.logger.Info("\n⭐ TOP 5 HIGHEST RATED PROPERTIES:")
	for i, listing := range analytics.TopRated {
		s.logger.Info("\n   %d. %s", i+1, listing.Title)
		s.logger.Info("      Rating:    %.2f ⭐", listing.Rating)
		s.logger.Info("      Price:     $%.2f per night", listing.Price)
		s.logger.Info("      Location:  %s", listing.Location)
		s.logger.Info("      Bedrooms: %d | Bathrooms: %d | Guests: %d",
			listing.Bedrooms, listing.Bathrooms, listing.Guests)
	}
	s.logger.Info("")
	return nil
}

// PrintByLocation prints listings grouped by location
func (s *AnalyticsService) PrintByLocation() error {
	analytics, err := s.GetAnalytics()
	if err != nil {
		return err
	}

	s.logger.Info("\n LISTINGS BY LOCATION:")
	for location, count := range analytics.ListingsPerLocation {
		s.logger.Info("   %-35s %d properties", location, count)
	}
	s.logger.Info("")
	return nil
}
