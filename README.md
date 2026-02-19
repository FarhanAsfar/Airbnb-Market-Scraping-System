# Airbnb Market Scraping System

A web scraper for Airbnb listings built with Go and chromedp. Automatically scrapes property data from multiple locations, stores in PostgreSQL, and provides analytics.

--- 

## Features

- **Multi-Location Scraping**: Automatically discovers and scrapes locations from Airbnb homepage
- **Detailed Property Data**: Title, price, location, rating, bedrooms, bathrooms, guest capacity, URL
- **Concurrent Scraping**: Worker pool pattern for parallel detail page scraping
- **Anti-Bot Detection**: 
  - Random delays between requests
  - User-agent rotation
  - Headless/headed browser modes
  - Rate limiting
- **Data Storage**: PostgreSQL with automatic deduplication
- **CSV Export**: Export all data to spreadsheet format
- **Analytics Dashboard**: Comprehensive statistics and insights
- **CLI Interface**: Multiple commands for different operations

---

## Prerequisites

Before you begin, ensure you have the following installed:

### Required Software

1. **Go 1.21 or higher**
   ```bash
   # Check Go version
   go version
   
   # If not installed, download from: https://go.dev/dl/
   ```

2. **Docker & Docker Compose**
   ```bash
   # Check Docker version
   docker --version
   docker-compose --version
   
   # If not installed:
   # Mac: brew install docker docker-compose
   # Ubuntu: sudo apt-get install docker.io docker-compose
   # Windows: Download Docker Desktop from docker.com
   ```

3. **Chrome or Chromium Browser**
   ```bash
   # Mac
   brew install --cask google-chrome
   
   # Ubuntu/Debian
   sudo apt-get install chromium-browser
   
   # Windows: Download from google.com/chrome
   
   # Verify installation
   google-chrome --version  # or chromium-browser --version
   ```

---

## Project Installation

### Step 1: Clone or Download the Project

```bash
# If using Git
git clone <your-repo-url>
cd airbnb-market-scraping-system

# Or extract the tar.gz file
tar -xzf airbnb-scraper.tar.gz
cd airbnb-scraper
```

### Step 2: Install Go Dependencies

```bash
# Initialize Go modules (if not already done)
go mod download

# This will download:
# - chromedp (browser automation)
# - lib/pq (PostgreSQL driver)
# - yaml.v3 (config parsing)
```

### Step 3: Start PostgreSQL Database

```bash
# Start PostgreSQL in Docker
docker-compose up -d

# Verify it's running
docker ps

# You should see: airbnb-postgres container running on port 5432

# Check logs if needed
docker-compose logs postgres
```

### Step 4: Verify Database Connection

```bash
# Connect to database
docker exec -it airbnb-postgres psql -U postgres -d airbnb_scraper

# You should see:
# airbnb_scraper=#

# Check tables (should see 'listings' table)
\dt

# Exit
\q
```

### Step 5: Configure the Scraper

Create a config.yaml file inside the config folder.

```bash
touch config/config.yaml
```
> !! For the assignment purpose I've pushed the file so that it would be less hassle to review the project. So, there is no need to create the file. Although this file doesn't contain any sensitive information, it is not the best practice.

**Key settings to review:**

```yaml
scraper:
  base_url: "https://www.airbnb.com"  # Don't change
  max_pages: 2                         # Pages per location
  properties_per_page: 5               # Properties per page
  max_workers: 3                       # Concurrent workers
  delay_min_ms: 2000                   # Min delay (increase if blocked)
  delay_max_ms: 5000                   # Max delay (increase if blocked)
  headless: false                      # true = no browser window
  
database:
  host: "localhost"
  port: 5432
  user: "postgres"                     # Change if you modified docker-compose
  password: "postgres"                 # Change if you modified docker-compose
  dbname: "airbnb_scraper"
```

### Step 6: Test Installation

```bash
# Run a quick test
go run main.go --show-stats

# If database is empty, it should show:
# Total listings: 0

# If you see this without errors, installation is successful! ✅
```
---

## ⚙️ Configuration

### Database Configuration

**Using default Docker setup** (recommended):
- The `docker-compose.yml` already sets up PostgreSQL
- Default credentials: `postgres:postgres`
- Database name: `airbnb_scraper`
- Port: `5432`

---

### Scraper Configuration

**For aggressive scraping** (risk of being blocked):
```yaml
delay_min_ms: 1000
delay_max_ms: 2000
max_workers: 5
headless: true
```

**For safe scraping** (recommended):
```yaml
delay_min_ms: 3000
delay_max_ms: 7000
max_workers: 2
headless: false
```

**If you get blocked** (503 errors):
```yaml
delay_min_ms: 5000
delay_max_ms: 10000
max_workers: 2
headless: true
max_retries: 5
```

---

### Full Scraping Workflow

```bash
# Run the complete scraping process
go run main.go
```

**This will:**
1. Visit Airbnb homepage
2. Extract all visible location cards
3. For each location:
   - Scrape page 1 (first 5 properties)
   - Scrape page 2 (first 5 properties)
4. Scrape detail pages (bedrooms, bathrooms, guests)
5. Save to PostgreSQL database
6. Export to CSV file (`listings.csv`)
7. Display analytics summary

**Expected runtime**: 5-15 minutes (depends on number of locations and delays)

### Headless Mode (No Browser Window)

```bash
# Edit config first
# Set headless: true in config/config.yaml

go run main.go
```

### Watch the Browser (Debug Mode)

```bash
# Set headless: false in config/config.yaml
go run main.go

# You'll see Chrome windows opening and navigating
# Useful for debugging or understanding the process
```

##  CLI Commands

### Analytics Commands (No Scraping)

**View all statistics:**
```bash
go run main.go --show-stats
```
Output:
```
 TOTAL LISTINGS: 50

 PRICE STATISTICS:
   Average Price: $156.32
   Maximum Price: $450.00
   Minimum Price: $45.00

 MOST EXPENSIVE PROPERTY:
   Title: Luxury Harbour View Apartment
   Price: $450.00 per night
   Location: Sydney
   Rating: 4.95
   Bedrooms: 3 | Bathrooms: 2 | Guests: 6

 LISTINGS PER LOCATION:
   Sydney: 10 listings
   Paris: 10 listings
   Tokyo: 10 listings
   ...

 TOP 5 HIGHEST RATED PROPERTIES:
   1. Cozy Studio in CBD
      Rating: 4.98 ⭐ | Price: $120.00 | Location: Sydney
   ...
```

**Specific statistics:**

```bash
# Average price only
go run main.go --avg-price

# Most expensive property
go run main.go --max-price

# Top 5 rated properties
go run main.go --top-rated

# Listings grouped by location
go run main.go --by-location
```

### Export Commands

```bash
# Export current database to CSV
go run main.go --export-csv

# Output: listings.csv
```

## 📁 Project Structure

```
airbnb-market-scraping-system/
├── config/
│   ├── config.yaml           # Configuration file
│   └── config.go             # Config loader
├── models/
│   └── listing.go            # Data models
├── scraper/
│   └── airbnb/
│       ├── scraper.go        # Main scraping logic
│       ├── detail_scraper.go # Detail page scraping
│       └── homepage_scraper.go # Homepage location extraction
├── storage/
│   ├── db.go                 # Database operations
│   └── schema.go             # SQL schema
├── services/
│   ├── listing_service.go    # Business logic
│   ├── analytics_service.go  # Analytics calculations
│   └── csv_service.go        # CSV export
├── utils/
│   ├── logger.go             # Logging utility
│   ├── normalize.go          # Data normalization
│   └── url.go                # URL utilities
├── docker-compose.yml        # PostgreSQL setup
├── go.mod                    # Go dependencies
├── main.go                   # Entry point
└── README.md                 # This file
```

##  How It Works

### 1. Homepage Location Discovery

```
Airbnb Homepage → Extract location cards → Get URLs for each location
Example: Sydney, Paris, Tokyo, Melbourne, Bangkok, etc.
```

### 2. Property Scraping

```
For each location:
  Page 1 → Extract 20 cards → Take first 5
  Page 2 → Extract 20 cards → Take first 5
  Total: 10 properties per location
```

### 3. Detail Page Scraping (Concurrent)

```
3 Workers process detail pages in parallel:
  Worker 1: Property 1, 4, 7, 10...
  Worker 2: Property 2, 5, 8, 11...
  Worker 3: Property 3, 6, 9, 12...

Each worker:
  - Visits detail page
  - Extracts bedrooms, bathrooms, guests
  - Retries up to 3 times on failure
```

### 4. Data Processing

```
Raw Data → Normalization → Database Storage
  - Price: "$120" → 120.00
  - Rating: "4.95 (123 reviews)" → 4.95
  - URL: Remove query params for deduplication
  - Location: Extract from title
```

### 5. Deduplication

```
Unique constraint on URL prevents duplicates
If URL exists: UPDATE existing record
If URL new: INSERT new record
```

## 🐛 Troubleshooting

### Issue: "503 Service Unavailable" or No Listings Found

**Cause**: Airbnb has temporarily blocked your IP due to too many requests.

**Solutions**:
1. **Wait 30-60 minutes** before trying again
2. **Increase delays** in `config/config.yaml`:
   ```yaml
   delay_min_ms: 5000  # 5 seconds
   delay_max_ms: 10000 # 10 seconds
   ```
3. **Run in headless mode**: Set `headless: true`
4. **Reduce workers**: Set `max_workers: 2`
5. **Use VPN or different network** if persistent

### Issue: "Failed to connect to database"

**Solutions**:
```bash
# Check if PostgreSQL is running
docker ps

# If not running, start it
docker-compose up -d

# Check logs
docker-compose logs postgres

# Restart containers
docker-compose restart

# If still fails, recreate containers
docker-compose down -v
docker-compose up -d
```

### Issue: "Chrome not found" or Browser Errors

**Solutions**:
```bash
# Mac
brew install --cask google-chrome

# Ubuntu
sudo apt-get install chromium-browser

# Verify installation
which google-chrome
which chromium-browser
```

### Issue: Duplicate Data in Database

**Cause**: URL normalization isn't working or Airbnb changed URL structure.

**Check**:
```sql
-- Connect to database
docker exec -it airbnb-postgres psql -U postgres -d airbnb_scraper

-- Check for duplicates
SELECT url, COUNT(*) FROM listings GROUP BY url HAVING COUNT(*) > 1;

-- If duplicates exist, clear and re-scrape
TRUNCATE listings RESTART IDENTITY;
```

### Issue: Title and Location Are the Same

**Cause**: Airbnb changed their HTML structure.


### Issue: Detail Pages Timeout

**Cause**: Detail pages take longer than 60 seconds to load.

**Solution**: Increase timeout in `scraper/airbnb/detail_scraper.go`:
```go
browserCtx, cancel = context.WithTimeout(browserCtx, 90*time.Second)
```

### Issue: Memory Issues During Scraping

**Solutions**:
1. Reduce `max_workers` to 2
2. Enable headless mode: `headless: true`
3. Reduce pages: `max_pages: 1`
4. Close other applications

---


##  Best Practices

### For Ethical Scraping

1. **Respect robots.txt**: Be polite, use reasonable delays
2. **Rate limiting**: Don't overwhelm Airbnb's servers
3. **User-agent**: Use realistic user agent strings
4. **Off-peak hours**: Run scraper during low-traffic times
5. **Personal use**: Don't republish or sell scraped data
6. **Check terms**: Review Airbnb's Terms of Service


---

Remember: Use responsibly and ethically. This tool is for educational and personal use only.


