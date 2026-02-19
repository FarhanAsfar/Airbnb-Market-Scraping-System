package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Config holds all configuration settings
type Config struct {
	Scraper  ScraperConfig  `yaml:"scraper"`
	Database DatabaseConfig `yaml:"database"`
	Output   OutputConfig   `yaml:"output"`
}

type ScraperConfig struct {
	BaseURL           string `yaml:"base_url"`
	MaxPages          int    `yaml:"max_pages"`
	PropertiesPerPage int    `yaml:"properties_per_page"`
	MaxWorkers        int    `yaml:"max_workers"`
	DelayMinMs        int    `yaml:"delay_min_ms"`
	DelayMaxMs        int    `yaml:"delay_max_ms"`
	MaxRetries        int    `yaml:"max_retries"`
	RetryDelayMs      int    `yaml:"retry_delay_ms"`
	Headless          bool   `yaml:"headless"`
	TimeoutSeconds    int    `yaml:"timeout_seconds"`
}

type DatabaseConfig struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
	DBName   string `yaml:"dbname"`
	SSLMode  string `yaml:"sslmode"`
}

type OutputConfig struct {
	CSVFile     string `yaml:"csv_file"`
	JSONConsole bool   `yaml:"json_console"`
}

// Load reads and parses the config file
func Load(filepath string) (*Config, error) {
	data, err := os.ReadFile(filepath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// Validate required fields
	if cfg.Scraper.BaseURL == "" {
		return nil, fmt.Errorf("scraper.url is required")
	}

	return &cfg, nil
}

// GetDSN returns PostgreSQL connection string
func (c *DatabaseConfig) GetDSN() string {
	return fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		c.Host, c.Port, c.User, c.Password, c.DBName, c.SSLMode)
}
