package utils

import (
	"regexp"
	"strconv"
	"strings"
)

// NormalizePrice extracts numeric price from strings like "$120", "£150 AUD", "$1,234"
// Returns the price as float64 or 0.0 if parsing fails
func NormalizePrice(raw string) float64 {
	// Extract all digits including decimal point
	re := regexp.MustCompile(`[\d,]+\.?\d*`)
	match := re.FindString(raw)

	if match == "" {
		return 0.0
	}

	// Remove commas
	match = strings.ReplaceAll(match, ",", "")

	price, err := strconv.ParseFloat(match, 64)
	if err != nil {
		return 0.0
	}

	return price
}

// NormalizeRating extracts numeric rating from strings like "4.95 (123 reviews)", "4.8"
// Returns the rating as float64 or 0.0 if parsing fails
func NormalizeRating(raw string) float64 {
	// Extract first decimal number
	re := regexp.MustCompile(`\d+\.?\d*`)
	match := re.FindString(raw)

	if match == "" {
		return 0.0
	}

	rating, err := strconv.ParseFloat(match, 64)
	if err != nil {
		return 0.0
	}

	// ceiling at 5.0
	if rating > 5.0 {
		rating = 5.0
	}

	return rating
}

// ExtractNumber extracts first integer from string
// Used for bedrooms, bathrooms, guests (e.g., "3 bedrooms" -> 3)
func ExtractNumber(raw string) int {
	re := regexp.MustCompile(`\d+`)
	match := re.FindString(raw)

	if match == "" {
		return 0
	}

	num, err := strconv.Atoi(match)
	if err != nil {
		return 0
	}

	return num
}

// CleanText removes extra whitespace and trims string
func CleanText(text string) string {
	// Remove leading/trailing whitespace
	text = strings.TrimSpace(text)

	// Replace multiple spaces with single space
	re := regexp.MustCompile(`\s+`)
	text = re.ReplaceAllString(text, " ")

	return text
}
