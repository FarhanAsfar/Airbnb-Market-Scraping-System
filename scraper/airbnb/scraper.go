package airbnb

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/chromedp/chromedp"
	"github.com/farhanasfar/airbnb-market-scraping-system/config"
	"github.com/farhanasfar/airbnb-market-scraping-system/models"
	"github.com/farhanasfar/airbnb-market-scraping-system/utils"
)

// Scraper handles Airbnb scraping operations
type Scraper struct {
	cfg    *config.ScraperConfig
	logger *utils.Logger
}

// NewScraper creates a new Airbnb scraper instance
func NewScraper(cfg *config.ScraperConfig, logger *utils.Logger) *Scraper {
	return &Scraper{
		cfg:    cfg,
		logger: logger,
	}
}

// createStealthContext creates a browser context with anti-detection settings
func (s *Scraper) createStealthContext(parentCtx context.Context) (context.Context, context.CancelFunc) {
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", s.cfg.Headless),
		chromedp.WindowSize(1440, 900),
		chromedp.UserAgent("Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"),
		chromedp.Flag("disable-blink-features", "AutomationControlled"),
		chromedp.Flag("blink-settings", "imagesEnabled=false"),
	)

	allocCtx, cancelAlloc := chromedp.NewExecAllocator(parentCtx, opts...)
	ctx, cancelCtx := chromedp.NewContext(allocCtx)

	return ctx, func() {
		cancelCtx()
		cancelAlloc()
	}
}

// removeWebdriverProperty removes the webdriver property that sites check
func removeWebdriverProperty() chromedp.Action {
	return chromedp.ActionFunc(func(ctx context.Context) error {
		err := chromedp.Evaluate(`
			Object.defineProperty(navigator, 'webdriver', {
				get: () => undefined
			})
		`, nil).Do(ctx)
		return err
	})
}

// ScrapeListings scrapes listings from a specific location URL
// Returns first PropertiesPerPage listings from each of MaxPages pages
// Returns first PropertiesPerPage listings from each of MaxPages pages
func (s *Scraper) ScrapeListings(ctx context.Context, locationURL string) ([]models.RawListing, error) {
	browserCtx, cancel := s.createStealthContext(ctx)
	defer cancel()

	s.logger.Info("Scraping location: %s", locationURL)
	s.logger.Info("Target: %d properties per page × %d pages = %d total",
		s.cfg.PropertiesPerPage, s.cfg.MaxPages, s.cfg.PropertiesPerPage*s.cfg.MaxPages)

	allListings := []models.RawListing{}

	// Navigate to first page
	err := chromedp.Run(browserCtx,
		removeWebdriverProperty(),
		chromedp.Navigate(locationURL),
		chromedp.WaitVisible(`[data-testid="card-container"]`, chromedp.ByQuery),
		chromedp.Sleep(3*time.Second),
	)

	if err != nil {
		return nil, fmt.Errorf("failed to load first page: %w", err)
	}

	// Scrape multiple pages
	for page := 1; page <= s.cfg.MaxPages; page++ {
		s.logger.Info("Scraping page %d/%d...", page, s.cfg.MaxPages)

		// Extract listings using inline JavaScript
		var listingsJSON string
		err := chromedp.Run(browserCtx,
			chromedp.Sleep(2*time.Second),
			chromedp.Evaluate(`
				JSON.stringify(
					Array.from(document.querySelectorAll('[data-testid="card-container"]')).slice(0, 20).map(card => {
						const getText = (selector) => {
							const el = card.querySelector(selector);
							return el ? el.innerText.trim() : '';
						};
						const getAttr = (selector, attr) => {
							const el = card.querySelector(selector);
							return el ? el.getAttribute(attr) : '';
						};
						return {
							title: getText('[data-testid="listing-card-title"]') || 
							       getText('[itemprop="name"]') ||
							       getText('div[id*="title"]'),
							price: getText('[data-testid="price-availability-row"]') ||
							       getText('span._tyxjp1') ||
							       getText('span[aria-label*="price"]'),
							location: getText('[data-testid="listing-card-subtitle"]') ||
							         getText('span[data-testid="listing-card-name"]'),
							rating: getAttr('[aria-label*="rating"]', 'aria-label') ||
							       getText('span[aria-label*="rating"]'),
							url: card.querySelector('a') ? card.querySelector('a').href : '',
							bedrooms: 0,
							bathrooms: 0,
							guests: 0
						};
					})
				)
			`, &listingsJSON),
		)

		if err != nil {
			s.logger.Error("Failed to extract listings from page %d: %v", page, err)
			continue
		}

		listings := s.parseListingsJSON(listingsJSON)

		// Limit to PropertiesPerPage (first 5)
		if len(listings) > s.cfg.PropertiesPerPage {
			listings = listings[:s.cfg.PropertiesPerPage]
		}

		s.logger.Success("Scraped %d listings from page %d (limited to first %d)",
			len(listings), page, s.cfg.PropertiesPerPage)
		allListings = append(allListings, listings...)

		// Navigate to next page if not last
		if page < s.cfg.MaxPages {
			time.Sleep(2 * time.Second)

			// Check for next button and click - ALL IN ONE chromedp.Run
			var hasNext bool
			s.logger.Info("Looking for 'Next' button...")

			err = chromedp.Run(browserCtx,
				// Check if next button exists
				chromedp.Evaluate(`
					(() => {
						const nextButton = document.querySelector('a[aria-label="Next"]') ||
						                   document.querySelector('a[aria-label*="next"]') ||
						                   document.querySelector('nav a:last-child');
						return nextButton && !nextButton.getAttribute('aria-disabled');
					})()
				`, &hasNext),
			)

			if err != nil {
				s.logger.Error("Failed to check for next button: %v", err)
				break
			}

			if !hasNext {
				s.logger.Info("No 'Next' button found, stopping at page %d", page)
				break
			}

			// Click next button
			err = chromedp.Run(browserCtx,
				chromedp.Click(`a[aria-label="Next"]`, chromedp.ByQuery),
			)

			if err != nil {
				// Try alternative selector
				s.logger.Warning("First selector failed, trying alternative...")
				err = chromedp.Run(browserCtx,
					chromedp.Click(`nav a:last-child`, chromedp.ByQuery),
				)
				if err != nil {
					s.logger.Error("Failed to click next button: %v", err)
					break
				}
			}

			s.logger.Success("Clicked 'Next' button")

			// Wait for new page to load
			err = chromedp.Run(browserCtx,
				chromedp.Sleep(3*time.Second),
				chromedp.WaitVisible(`[data-testid="card-container"]`, chromedp.ByQuery),
			)

			if err != nil {
				s.logger.Error("Failed waiting for next page: %v", err)
				break
			}

			s.logger.Success("Page %d loaded", page+1)
		}
	}

	s.logger.Success("Total listings scraped for this location: %d", len(allListings))
	return allListings, nil
}

// goToNextPage clicks the pagination next button
func (s *Scraper) goToNextPage(ctx context.Context) (bool, error) {
	s.logger.Info("Looking for 'Next' button...")

	var nextButtonExists bool
	err := chromedp.Evaluate(`
		(() => {
			const nextButton = document.querySelector('a[aria-label="Next"]') ||
			                   document.querySelector('a[aria-label*="next"]') ||
			                   document.querySelector('nav a:last-child');
			return nextButton && !nextButton.getAttribute('aria-disabled');
		})()
	`, &nextButtonExists).Do(ctx)

	if err != nil {
		return false, fmt.Errorf("failed to check for next button: %w", err)
	}

	if !nextButtonExists {
		return false, nil
	}

	// Click next button
	err = chromedp.Click(`a[aria-label="Next"]`, chromedp.ByQuery).Do(ctx)
	if err != nil {
		// Try alternative
		err = chromedp.Click(`nav a:last-child`, chromedp.ByQuery).Do(ctx)
		if err != nil {
			return false, fmt.Errorf("failed to click next button: %w", err)
		}
	}

	s.logger.Success("Navigated to next page")
	return true, nil
}

// parseListingsJSON parses JSON into RawListing structs
func (s *Scraper) parseListingsJSON(jsonStr string) []models.RawListing {
	if jsonStr == "" || jsonStr == "[]" || jsonStr == "null" {
		return []models.RawListing{}
	}

	var rawData []struct {
		Title     string `json:"title"`
		Price     string `json:"price"`
		Location  string `json:"location"`
		Rating    string `json:"rating"`
		URL       string `json:"url"`
		Bedrooms  int    `json:"bedrooms"`
		Bathrooms int    `json:"bathrooms"`
		Guests    int    `json:"guests"`
	}

	if err := json.Unmarshal([]byte(jsonStr), &rawData); err != nil {
		s.logger.Error("Failed to parse JSON: %v", err)
		return []models.RawListing{}
	}

	listings := make([]models.RawListing, 0, len(rawData))
	for _, raw := range rawData {
		listing := models.RawListing{
			Title:     utils.CleanText(raw.Title),
			Price:     utils.CleanText(raw.Price),
			Location:  utils.CleanText(raw.Location),
			Rating:    utils.CleanText(raw.Rating),
			URL:       raw.URL,
			Bedrooms:  raw.Bedrooms,
			Bathrooms: raw.Bathrooms,
			Guests:    raw.Guests,
		}

		if listing.Title != "" && listing.URL != "" {
			listings = append(listings, listing)
		}
	}

	return listings
}
