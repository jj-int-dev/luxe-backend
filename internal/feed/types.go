// Package feed owns the feed generation layer:
// filtering, seen_ids exclusion, deterministic shuffle, and cursor pagination.
//
// Architecture:  itemhandler → feed → item
//
// feed owns the DTOs (GetFeedInput / GetFeedOutput) and the Service interface.
// The item package remains a pure domain package (Item, Repository, errors).
package feed

import (
	"context"

	"luxe-backend/internal/item"
)

// GetFeedInput carries the parameters for a feed request.
type GetFeedInput struct {
	Category *string
	Cursor   *string
	Limit    int
	SeenIDs  []string
}

// GetFeedOutput carries the result of a feed request.
type GetFeedOutput struct {
	Items      []item.Item
	NextCursor *string
}

// Service defines the feed generation contract.
type Service interface {
	GetFeed(ctx context.Context, input GetFeedInput) (GetFeedOutput, error)
}

// Re-export the domain sentinel errors so that handler packages (e.g.
// internal/itemhandler) only need to import feed, not item directly.
var (
	ErrInvalidCategory = item.ErrInvalidCategory
	ErrInvalidCursor   = item.ErrInvalidCursor
	ErrInvalidLimit    = item.ErrInvalidLimit
)
