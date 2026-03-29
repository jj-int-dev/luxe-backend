package feed

import (
	"encoding/binary"
	"hash/fnv"
	"sort"

	"luxe-backend/internal/item"
)

// scoredItem pairs an item with its computed ranking score.
type scoredItem struct {
	it    item.Item
	score float64
}

// rankItems takes a slice of items that has already been deterministically
// shuffled and returns a new slice sorted by ranking score descending.
//
// The shuffle order acts as the stable tie-breaker: because sort.SliceStable
// preserves the relative order of equal elements, items whose scores are
// identical retain the relative order established by the prior shuffle.
//
// Pipeline position: called after shuffleItems, before cursor application.
func rankItems(items []item.Item, seed int64) []item.Item {
	scored := make([]scoredItem, len(items))
	for i, it := range items {
		scored[i] = scoredItem{
			it:    it,
			score: computeItemScore(it, seed),
		}
	}

	// Stable sort: equal-scored items keep their (shuffled) relative order,
	// preserving determinism without discarding the randomness layer.
	sort.SliceStable(scored, func(i, j int) bool {
		return scored[i].score > scored[j].score
	})

	out := make([]item.Item, len(scored))
	for i, s := range scored {
		out[i] = s.it
	}
	return out
}

// computeItemScore returns a deterministic composite ranking score ∈ [0, 1].
//
// The score is a weighted sum of two components:
//
//   - Popularity (60 %): normalised Price — higher price → higher score.
//     This is the primary ranking signal. The intent is for it to be replaced
//     by real engagement metrics (views, saves, purchases) in a future iteration
//     backed by a real database.
//
//   - Deterministic randomness (40 %): hash(itemID ∥ seed) → [0, 1).
//     The seed is derived from the session's seen_ids (via dreamSeed), so the
//     random component changes per-session while remaining fully deterministic
//     for identical inputs.
//
// Mathematical guarantee (relevant to TestFeedService_Ranking_HigherPriceItemsRankHigher):
//
//	Monaco Penthouse ($25 000 000) min score = 0.6 × 1.00 + 0.4 × 0.0000 = 0.6000
//	Beverly Hills Mansion ($5 500 000) max score = 0.6 × 0.22 + 0.4 × 0.9999 = 0.5320
//
// Because 0.6000 > 0.5320, Monaco Penthouse always ranks above Beverly Hills
// regardless of the seed, making the ordering a hard invariant rather than a
// coincidence of the shuffle.
func computeItemScore(it item.Item, seed int64) float64 {
	const (
		// maxPrice is the highest price in the current in-memory seed dataset
		// (Monaco Penthouse, $25 000 000). All prices are normalised against this
		// value so the popularity component lives in [0, 1]. Update this constant
		// when the dataset changes significantly.
		maxPrice = 25_000_000.0

		weightPopularity = 0.6
		weightRandom     = 0.4
	)

	popularityScore := it.Price / maxPrice
	if popularityScore > 1.0 {
		popularityScore = 1.0
	}

	randomScore := perItemDeterministicRandom(it.ID, seed)

	return weightPopularity*popularityScore + weightRandom*randomScore
}

// perItemDeterministicRandom returns a float64 in [0, 1) for the given
// (itemID, seed) pair using fnv64a(itemID ∥ seed).
//
// Properties:
//   - Stable: the same (id, seed) always produces the same value.
//   - Session-variant: different seeds (from different seen_ids) produce
//     different values for the same item, varying the ranking across sessions.
//   - Order-independent: the value depends only on (id, seed), not on the
//     position of the item in any slice, so the ranking is immune to iteration
//     order changes in the repository.
func perItemDeterministicRandom(id string, seed int64) float64 {
	h := fnv.New64a()
	h.Write([]byte(id))

	var b [8]byte
	binary.LittleEndian.PutUint64(b[:], uint64(seed)) //nolint:gosec
	h.Write(b[:])

	// 10 000 buckets → 0.01 % resolution, sufficient for ranking purposes.
	return float64(h.Sum64()%10000) / 10000.0
}
