package storage

const (
	// creating the listings table with indexes
	CreateListingsTableSQL = `
	CREATE TABLE IF NOT EXISTS listings (
		id SERIAL PRIMARY KEY,
		title TEXT NOT NULL,
		price DECIMAL(10, 2) NOT NULL,
		location TEXT NOT NULL,
		rating DECIMAL(3, 2) DEFAULT 0.0,
		url TEXT UNIQUE NOT NULL,  -- Ensures no duplicate listings
		bedrooms INTEGER DEFAULT 0,
		bathrooms INTEGER DEFAULT 0,
		guests INTEGER DEFAULT 0,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);

	-- Index on price for analytics queries (avg, min, max)
	CREATE INDEX IF NOT EXISTS idx_listings_price ON listings(price);
	
	-- Index on location for location-based queries
	CREATE INDEX IF NOT EXISTS idx_listings_location ON listings(location);
	
	-- Index on rating for top-rated queries
	CREATE INDEX IF NOT EXISTS idx_listings_rating ON listings(rating DESC);
	
	-- Unique constraint on URL prevents duplicates
	CREATE UNIQUE INDEX IF NOT EXISTS idx_listings_url ON listings(url);
	`

	// UpdateUpdatedAtTriggerSQL creates a trigger to auto-update updated_at
	UpdateUpdatedAtTriggerSQL = `
	CREATE OR REPLACE FUNCTION update_updated_at_column()
	RETURNS TRIGGER AS $$
	BEGIN
		NEW.updated_at = CURRENT_TIMESTAMP;
		RETURN NEW;
	END;
	$$ language 'plpgsql';

	DROP TRIGGER IF EXISTS update_listings_updated_at ON listings;

	CREATE TRIGGER update_listings_updated_at
		BEFORE UPDATE ON listings
		FOR EACH ROW
		EXECUTE FUNCTION update_updated_at_column();
	`
)
