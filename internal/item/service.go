package item

import (
	"context"
	"hash/fnv"
	"math/rand"
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
	// seenIDs  – IDs to exclude from the result set (Dream Mode feed).
	//
	// nextCursor is nil when the caller has reached the last page.
	GetItems(ctx context.Context, category *string, cursor *string, limit int, seenIDs []string) ([]Item, *string, error)
}

type service struct {
	repo Repository
}

// NewService returns a Service backed by the given Repository.
func NewService(repo Repository) Service {
	return &service{repo: repo}
}

// dreamSeed builds a deterministic int64 seed from the category and seenIDs.
//
// seenIDs are sorted before hashing so that the order in which the caller
// provides them does not affect the seed (and therefore the shuffle).
// The seed is intentionally independent of the cursor so that cursor-based
// pagination remains consistent across pages with the same seen_ids.
func dreamSeed(category *string, seenIDs []string) int64 {
	var sb strings.Builder
	if category != nil {
		sb.WriteString(*category)
	}
	sb.WriteByte('|')

	sorted := make([]string, len(seenIDs))
	copy(sorted, seenIDs)
	sort.Strings(sorted)
	sb.WriteString(strings.Join(sorted, ","))

	h := fnv.New64a()
	h.Write([]byte(sb.String()))
	return int64(h.Sum64())
}

// shuffleItems returns a new slice with items shuffled deterministically using
// the given seed. The caller must ensure the input is in a stable order
// (sorted by ID) before calling so that the same seed always produces the same
// output regardless of the underlying repository iteration order.
func shuffleItems(items []Item, seed int64) []Item {
	out := make([]Item, len(items))
	copy(out, items)
	//nolint:gosec // math/rand is intentional: we want fast, seeded, non-crypto randomness.
	rng := rand.New(rand.NewSource(seed)) //nolint:gosec
	rng.Shuffle(len(out), func(i, j int) { out[i], out[j] = out[j], out[i] })
	return out
}

// GetItems implements Service.
func (s *service) GetItems(ctx context.Context, category *string, cursor *string, limit int, seenIDs []string) ([]Item, *string, error) {
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

	// ── 4. Sort deterministically by ID (stable baseline for shuffle) ────────
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

	// ── 6. Exclude seen IDs (Dream Mode) ─────────────────────────────────────
	if len(seenIDs) > 0 {
		seenSet := make(map[string]bool, len(seenIDs))
		for _, id := range seenIDs {
			seenSet[id] = true
		}
		filtered := make([]Item, 0, len(all))
		for _, it := range all {
			if !seenSet[it.ID] {
				filtered = append(filtered, it)
			}
		}
		all = filtered
	}

	// ── 7. Return empty when nothing remains ─────────────────────────────────
	if len(all) == 0 {
		return []Item{}, nil, nil
	}

	// ── 8. Shuffle deterministically ─────────────────────────────────────────
	// Seed is derived from category + seenIDs only (not cursor) so that
	// cursor-based pagination within the same seen_ids session remains stable.
	seed := dreamSeed(category, seenIDs)
	all = shuffleItems(all, seed)

	// ── 9. Apply cursor (items strictly after the cursor position) ───────────
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

	// ── 10. Slice to limit and calculate next cursor ──────────────────────────
	var nextCursor *string
	if len(all) > limit {
		nc := all[limit-1].ID
		nextCursor = &nc
		all = all[:limit]
	}

	return all, nextCursor, nil
}
