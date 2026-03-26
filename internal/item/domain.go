package item

import "errors"

// Item is the core domain model for a luxury item.
type Item struct {
	ID       string
	Title    string
	Price    float64
	ImageURL string
	Category string
}

// validCategories is the exhaustive set of accepted category values.
var validCategories = map[string]bool{
	"cars":    true,
	"houses":  true,
	"jewelry": true,
}

// ErrInvalidCategory is returned when an unrecognised category is requested.
var ErrInvalidCategory = errors.New("category must be one of: cars, houses, jewelry")

// IsValidCategory reports whether the given category string is allowed.
func IsValidCategory(c string) bool {
	return validCategories[c]
}
