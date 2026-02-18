package airbnb

import (
	"context"

	"github.com/chromedp/chromedp"
	"github.com/farhanasfar/airbnb-market-scraping-system/config"
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
