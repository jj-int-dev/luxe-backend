package feed

import (
	"context"
	"hash/fnv"
	"math/rand"
	"sort"
	"strings"

	"luxe-backend/internal/item"
)

// Compile-time assertion: FeedService must implement Service.
var _ Service = (*FeedService)(nil)

const maxLimit = 50

// FeedService is the canonical feed generation implementation.
// It is the single authoritative source of feed logic:
//
//  1. validate limit & category
//  2. fetch all items
//  3. sort by ID (stable baseline)
//  4. filter by category
//  5. exclude seen_ids
//  6. shuffle deterministically (seed = hash(category + sorted(seen_ids)))
//  7. apply cursor
//  8. apply limit / compute next cursor
type FeedService struct {
	repo item.Repository
}

// NewFeedService returns a FeedService backed by the given Repository.
func NewFeedService(repo item.Repository) *FeedService {
	return &FeedService{repo: repo}
}

// dreamSeed builds a deterministic int64 seed from the (already-normalised)
// category and seenIDs.
//
// seenIDs are sorted before hashing so that the order in which the caller
// provides them does not affect the seed (and therefore the shuffle).
// The seed is intentionally independent of the cursor so that cursor-based
// pagination remains stable across pages that share the same seen_ids.
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
// output regardless of underlying repository iteration order.
func shuffleItems(items []item.Item, seed int64) []item.Item {
	out := make([]item.Item, len(items))
	copy(out, items)
	//nolint:gosec // math/rand is intentional: fast, seeded, non-crypto randomness.
	rng := rand.New(rand.NewSource(seed)) //nolint:gosec
	rng.Shuffle(len(out), func(i, j int) { out[i], out[j] = out[j], out[i] })
	return out
}

// GetFeed implements Service.
func (s *FeedService) GetFeed(ctx context.Context, input GetFeedInput) (GetFeedOutput, error) {
	// ── 1. Validate limit ────────────────────────────────────────────────────
	if input.Limit <= 0 {
		return GetFeedOutput{}, item.ErrInvalidLimit
	}
	if input.Limit > maxLimit {
		input.Limit = maxLimit
	}

	// ── 2. Validate and normalise category ───────────────────────────────────
	// Use a local variable; never mutate the caller's *string.
	var normalizedCategory *string
	if input.Category != nil {
		nc := strings.ToLower(*input.Category)
		if !item.IsValidCategory(nc) {
			return GetFeedOutput{}, item.ErrInvalidCategory
		}
		normalizedCategory = &nc
	}

	// ── 3. Fetch all items ───────────────────────────────────────────────────
	all, err := s.repo.GetAll(ctx)
	if err != nil {
		return GetFeedOutput{}, err
	}

	// ── 4. Sort deterministically by ID (stable baseline for shuffle) ────────
	sort.Slice(all, func(i, j int) bool {
		return all[i].ID < all[j].ID
	})

	// ── 5. Filter by category ────────────────────────────────────────────────
	if normalizedCategory != nil {
		filtered := make([]item.Item, 0, len(all))
		for _, it := range all {
			if it.Category == *normalizedCategory {
				filtered = append(filtered, it)
			}
		}
		all = filtered
	}

	// ── 6. Exclude seen IDs (Dream Mode) ─────────────────────────────────────
	if len(input.SeenIDs) > 0 {
		seenSet := make(map[string]bool, len(input.SeenIDs))
		for _, id := range input.SeenIDs {
			seenSet[id] = true
		}
		filtered := make([]item.Item, 0, len(all))
		for _, it := range all {
			if !seenSet[it.ID] {
				filtered = append(filtered, it)
			}
		}
		all = filtered
	}

	// ── 7. Return empty when nothing remains ─────────────────────────────────
	if len(all) == 0 {
		return GetFeedOutput{Items: []item.Item{}, NextCursor: nil}, nil
	}

	// ── 8. Shuffle deterministically ─────────────────────────────────────────
	// Seed is derived from normalised category + seenIDs only (not the cursor)
	// so that cursor-based pagination within the same seen_ids session is stable.
	seed := dreamSeed(normalizedCategory, input.SeenIDs)
	all = shuffleItems(all, seed)

	// ── 9. Apply cursor (items strictly after the cursor position) ───────────
	if input.Cursor != nil {
		idx := -1
		for i, it := range all {
			if it.ID == *input.Cursor {
				idx = i
				break
			}
		}
		if idx == -1 {
			return GetFeedOutput{}, item.ErrInvalidCursor
		}
		all = all[idx+1:]
	}

	// ── 10. Slice to limit and calculate next cursor ──────────────────────────
	var nextCursor *string
	if len(all) > input.Limit {
		nc := all[input.Limit-1].ID
		nextCursor = &nc
		all = all[:input.Limit]
	}

	return GetFeedOutput{Items: all, NextCursor: nextCursor}, nil
}
