package airbnb

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/chromedp/chromedp"
)

// DetailResult holds the result of scraping a detail page
type DetailResult struct {
	URL       string
	Bedrooms  int
	Bathrooms int
	Guests    int
	Error     error
}

// ScrapeDetailPage extracts bedroom, bathroom, and guest info from a listing detail page
func (s *Scraper) ScrapeDetailPage(ctx context.Context, url string) (*DetailResult, error) {
	result := &DetailResult{URL: url}

	// Create a new browser context for this detail page
	browserCtx, cancel := s.createStealthContext(ctx)
	defer cancel()

	// Add timeout for detail page
	browserCtx, cancel = context.WithTimeout(browserCtx, 30*time.Second)
	defer cancel()

	s.logger.Info("Scraping detail page: %s", url)

	// Add rate limiting delay before visiting
	// if err := chromedp.Sleep(2 * time.Second).Do(browserCtx); err != nil {
	// 	return result, fmt.Errorf("delay failed: %w", err)
	// }

	var detailsJSON string

	err := chromedp.Run(browserCtx,
		removeWebdriverProperty(),
		chromedp.Navigate(url),

		// Wait for the page to load - looking for common Airbnb detail page elements
		chromedp.WaitVisible(`[data-section-id="OVERVIEW_DEFAULT"]`, chromedp.ByQuery),

		// Extract details using JavaScript
		chromedp.Evaluate(`
			JSON.stringify({
				bedrooms: (() => {
					// Try multiple selectors for bedrooms
					const bedroomText = Array.from(document.querySelectorAll('li, span, div'))
						.map(el => el.innerText)
						.find(text => /(\d+)\s*(bedroom|bed)/i.test(text));
					if (bedroomText) {
						const match = bedroomText.match(/(\d+)\s*(bedroom|bed)/i);
						return match ? parseInt(match[1]) : 0;
					}
					return 0;
				})(),
				bathrooms: (() => {
					// Try multiple selectors for bathrooms
					const bathroomText = Array.from(document.querySelectorAll('li, span, div'))
						.map(el => el.innerText)
						.find(text => /(\d+\.?\d*)\s*bath/i.test(text));
					if (bathroomText) {
						const match = bathroomText.match(/(\d+\.?\d*)\s*bath/i);
						return match ? parseFloat(match[1]) : 0;
					}
					return 0;
				})(),
				guests: (() => {
					// Try multiple selectors for guests
					const guestText = Array.from(document.querySelectorAll('li, span, div'))
						.map(el => el.innerText)
						.find(text => /(\d+)\s*guest/i.test(text));
					if (guestText) {
						const match = guestText.match(/(\d+)\s*guest/i);
						return match ? parseInt(match[1]) : 0;
					}
					return 0;
				})()
			})
		`, &detailsJSON),
	)

	if err != nil {
		result.Error = fmt.Errorf("failed to scrape detail page: %w", err)
		return result, result.Error
	}

	// Parse the JSON response
	var details struct {
		Bedrooms  int     `json:"bedrooms"`
		Bathrooms float64 `json:"bathrooms"`
		Guests    int     `json:"guests"`
	}

	if err := json.Unmarshal([]byte(detailsJSON), &details); err != nil {
		result.Error = fmt.Errorf("failed to parse details JSON: %w", err)
		return result, result.Error
	}

	result.Bedrooms = details.Bedrooms
	result.Bathrooms = int(details.Bathrooms) // Convert to int for storage
	result.Guests = details.Guests

	s.logger.Success("Detail page scraped: %d beds, %d baths, %d guests",
		result.Bedrooms, result.Bathrooms, result.Guests)

	return result, nil
}

// ScrapeDetailsWithWorkers scrapes multiple detail pages concurrently using worker pool
func (s *Scraper) ScrapeDetailsWithWorkers(ctx context.Context, urls []string) map[string]*DetailResult {
	results := make(map[string]*DetailResult)
	resultsMux := &sync.Mutex{}

	// Create a channel for URLs to process
	urlChan := make(chan string, len(urls))

	// Create a wait group for workers
	var wg sync.WaitGroup

	// Start worker goroutines
	numWorkers := s.cfg.MaxWorkers
	if numWorkers <= 0 {
		numWorkers = 3 // Default
	}

	s.logger.Info("Starting %d workers for detail page scraping...", numWorkers)

	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()

			for url := range urlChan {
				s.logger.Info("[Worker %d] Processing: %s", workerID, url)

				// Scrape with retry logic
				result := s.scrapeDetailWithRetry(ctx, url)

				// Store result
				resultsMux.Lock()
				results[url] = result
				resultsMux.Unlock()
			}

			s.logger.Info("[Worker %d] Finished", workerID)
		}(i + 1)
	}

	// Send URLs to workers
	for _, url := range urls {
		urlChan <- url
	}
	close(urlChan)

	// Wait for all workers to finish
	wg.Wait()

	s.logger.Success("All detail pages scraped")
	return results
}

// scrapeDetailWithRetry attempts to scrape a detail page with retries
func (s *Scraper) scrapeDetailWithRetry(ctx context.Context, url string) *DetailResult {
	maxRetries := s.cfg.MaxRetries
	if maxRetries <= 0 {
		maxRetries = 3 // Default
	}

	var lastErr error
	var result *DetailResult

	for attempt := 1; attempt <= maxRetries; attempt++ {
		result, lastErr = s.ScrapeDetailPage(ctx, url)

		if lastErr == nil {
			return result // Success
		}

		// Log retry attempt
		if attempt < maxRetries {
			s.logger.Warning("Attempt %d/%d failed for %s: %v. Retrying...",
				attempt, maxRetries, url, lastErr)

			// Wait before retry
			time.Sleep(time.Duration(s.cfg.RetryDelayMs) * time.Millisecond)
		}
	}

	// All retries failed
	s.logger.Error("Failed to scrape %s after %d attempts: %v", url, maxRetries, lastErr)
	return &DetailResult{
		URL:   url,
		Error: fmt.Errorf("failed after %d retries: %w", maxRetries, lastErr),
	}
}
