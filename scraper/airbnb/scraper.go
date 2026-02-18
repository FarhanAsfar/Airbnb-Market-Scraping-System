package airbnb

import (
	"context"
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

	var listings []models.RawListing

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

		// Scroll to trigger lazy loading
		chromedp.Evaluate(`window.scrollTo(0, document.body.scrollHeight / 2)`, nil),
		scrape.randomDelay(),
		chromedp.Evaluate(`window.scrollTo(0, 0)`, nil),

		// Extract listings using JavaScript
		chromedp.ActionFunc(func(ctx context.Context) error {
			var err error
			listings, err = scrape.extractListings(ctx)
			return err
		}),
	)

	if err != nil {
		return nil, fmt.Errorf("scraping failed: %w", err)
	}

	scrape.logger.Success("Scraped %d listings from page", len(listings))
	return listings, nil
}
