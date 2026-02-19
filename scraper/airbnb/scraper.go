package airbnb

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
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
func (scrape *Scraper) createStealthContext(parentCtx context.Context) (context.Context, context.CancelFunc) {
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		// avoiding bot detection
		chromedp.Flag("headless", scrape.cfg.Headless),
		chromedp.WindowSize(1440, 900),
		chromedp.UserAgent("Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"),
		chromedp.Flag("disable-blink-features", "AutomationControlled"),
		chromedp.Flag("blink-settings", "imagesEnabled=false"), //not loading images to scrape fast.
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

// randomDelay adding human-like random delay
func (scrape *Scraper) randomDelay() chromedp.Action {
	return chromedp.ActionFunc(func(ctx context.Context) error {
		minMs := scrape.cfg.DelayMinMs
		maxMs := scrape.cfg.DelayMaxMs
		delay := time.Duration(minMs+rand.Intn(maxMs-minMs)) * time.Millisecond

		scrape.logger.Info("Waiting %v before next action...", delay)
		time.Sleep(delay)
		return nil
	})
}

// ScrapeListings scrapes listings from Airbnb search results
func (scrape *Scraper) ScrapeListings(ctx context.Context) ([]models.RawListing, error) {
	// Create stealth browser context
	browserCtx, cancel := scrape.createStealthContext(ctx)
	defer cancel()

	// Add timeout
	browserCtx, cancel = context.WithTimeout(browserCtx, time.Duration(scrape.cfg.TimeoutSeconds)*time.Second)
	defer cancel()

	scrape.logger.Info("Starting Airbnb scraper...")
	scrape.logger.Info("Target URL: %s", scrape.cfg.URL)
	scrape.logger.Info("Max Pages: %d", scrape.cfg.MaxPages)

	var allListings []models.RawListing

	//Navigate to first page
	err := chromedp.Run(browserCtx,
		// Remove webdriver property
		removeWebdriverProperty(),

		// Navigate to search page
		chromedp.Navigate(scrape.cfg.URL),

		// Waiting for listing cards to appear
		// Using data-testid attribute
		chromedp.WaitVisible(`[data-testid="card-container"]`, chromedp.ByQuery),

		// Add delay to let page fully render
		scrape.randomDelay(),
	)

	if err != nil {
		return nil, fmt.Errorf("failed to load first page: %w", err)
	}

	//scrape multiple pages
	for page := 1; page <= scrape.cfg.MaxPages; page++ {
		scrape.logger.Info("Scraping page %d/%d...", page, scrape.cfg.MaxPages)

		// Scroll to trigger lazy loading
		chromedp.Run(browserCtx,
			chromedp.Evaluate(`window.scrollTo(0, document.body.scrollHeight / 2)`, nil),
			scrape.randomDelay(),
			chromedp.Evaluate(`window.scrollTo(0, 0)`, nil),
		)

		// Extract listings from current page
		listings, err := scrape.extractListings(browserCtx)
		if err != nil {
			scrape.logger.Error("Failed to extract listings from page %d: %v", page, err)
			continue
		}

		scrape.logger.Success("Scraped %d listings from page %d", len(listings), page)
		allListings = append(allListings, listings...)

		// If not the last page, click "Next" button
		if page < scrape.cfg.MaxPages {
			hasNext, err := scrape.goToNextPage(browserCtx)
			if err != nil {
				scrape.logger.Error("Failed to navigate to next page: %v", err)
				break
			}

			if !hasNext {
				scrape.logger.Info("No more pages available, stopping at page %d", page)
				break
			}

			// Wait for new page to load
			chromedp.Run(browserCtx,
				chromedp.WaitVisible(`[data-testid="card-container"]`, chromedp.ByQuery),
				scrape.randomDelay(),
			)
		}

	}
	scrape.logger.Success("Scraped %d listings from page", len(allListings))
	return allListings, nil
}

// extractListings extracts listing data using JavaScript evaluation
func (scrape *Scraper) extractListings(ctx context.Context) ([]models.RawListing, error) {
	// JavaScript code to extract all listing data
	// We are using data-testid and aria-label attributes
	jsCode := `
		JSON.stringify(
			Array.from(document.querySelectorAll('[data-testid="card-container"]')).slice(0, 20).map(card => {
				// Helper function to safely get text content
				const getText = (selector) => {
					const el = card.querySelector(selector);
					return el ? el.innerText.trim() : '';
				};

				// Helper function to safely get attribute
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
	`

	var listingsJSON string
	if err := chromedp.Evaluate(jsCode, &listingsJSON).Do(ctx); err != nil {
		return nil, fmt.Errorf("failed to evaluate JS: %w", err)
	}

	// Convert JSON string to RawListing structs
	listings := scrape.parseListingsJSON(listingsJSON)

	//create a simple parser
	scrape.logger.Info("Extracted %d listing data from page", len(listings))

	return listings, nil
}

// parseListingsJSON parses the JSON response into RawListing structs
func (s *Scraper) parseListingsJSON(jsonStr string) []models.RawListing {
	// Handling empty responses
	if jsonStr == "" || jsonStr == "[]" || jsonStr == "null" {
		return []models.RawListing{}
	}

	// structure matching the JavaScript output
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

	// Parse JSON
	if err := json.Unmarshal([]byte(jsonStr), &rawData); err != nil {
		s.logger.Error("Failed to parse JSON: %v", err)
		return []models.RawListing{}
	}

	// Convert to RawListing structs
	listings := make([]models.RawListing, 0, len(rawData))
	for _, raw := range rawData {
		// Clean the data
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

		// Only add listings with valid data
		if listing.Title != "" && listing.URL != "" {
			listings = append(listings, listing)
		}
	}

	return listings
}
