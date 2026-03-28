package item

import (
	"context"
	"sort"
	"strings"
)

const maxLimit = 50

// Service defines the business-logic contract for items.
type Service interface {
	// GetItems returns a page of items and the next cursor.
	//
	// category – optional filter; ErrInvalidCategory if unrecognised.
	// cursor   – optional ID of the last item seen; ErrInvalidCursor if not found.
	// limit    – page size (1–50); ErrInvalidLimit if ≤ 0; clamped to 50 if > 50.
	//
	// nextCursor is nil when the caller has reached the last page.
	GetItems(ctx context.Context, category *string, cursor *string, limit int) ([]Item, *string, error)
}

type service struct {
	repo Repository
}

// NewService returns a Service backed by the given Repository.
func NewService(repo Repository) Service {
	return &service{repo: repo}
}

// GetItems implements Service.
func (s *service) GetItems(ctx context.Context, category *string, cursor *string, limit int) ([]Item, *string, error) {
	// ── 1. Validate limit ────────────────────────────────────────────────────
	if limit <= 0 {
		return nil, nil, ErrInvalidLimit
	}
	if limit > maxLimit {
		limit = maxLimit
	}

	// ── 2. Validate category ─────────────────────────────────────────────────
	if category != nil {
		normalized := strings.ToLower(*category)
		if !IsValidCategory(normalized) {
			return nil, nil, ErrInvalidCategory
		}
		*category = normalized
	}

	// ── 3. Fetch all items ───────────────────────────────────────────────────
	all, err := s.repo.GetAll(ctx)
	if err != nil {
		return nil, nil, err
	}

	// ── 4. Sort deterministically by ID (ascending) ──────────────────────────
	sort.Slice(all, func(i, j int) bool {
		return all[i].ID < all[j].ID
	})

	// ── 5. Filter by category ────────────────────────────────────────────────
	if category != nil {
		filtered := make([]Item, 0, len(all))
		for _, it := range all {
			if it.Category == *category {
				filtered = append(filtered, it)
			}
		}
		all = filtered
	}

	// ── 6. Apply cursor (items strictly after the cursor position) ───────────
	if cursor != nil {
		idx := -1
		for i, it := range all {
			if it.ID == *cursor {
				idx = i
				break
			}
		}
		if idx == -1 {
			return nil, nil, ErrInvalidCursor
		}
		all = all[idx+1:]
	}

	// ── 7. Slice to limit and calculate next cursor ──────────────────────────
	var nextCursor *string
	if len(all) > limit {
		nc := all[limit-1].ID
		nextCursor = &nc
		all = all[:limit]
	}

	return all, nextCursor, nil
}
