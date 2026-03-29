package feed_test

import (
	"context"
	"testing"

	"luxe-backend/internal/feed"
	"luxe-backend/internal/item"
)

// ── helpers ───────────────────────────────────────────────────────────────────

func strPtr(s string) *string { return &s }

// newSvc returns a feed.Service backed by the real in-memory repository
// (7 cars + 5 houses + 0 jewelry).
func newSvc() feed.Service {
	return feed.NewFeedService(item.NewInMemoryRepository())
}

func ctx() context.Context { return context.Background() }

// ── Error cases ───────────────────────────────────────────────────────────────

func TestFeedService_ErrorCases(t *testing.T) {
	svc := newSvc()

	tests := []struct {
		name    string
		input   feed.GetFeedInput
		wantErr error
	}{
		{
			name:    "zero limit → ErrInvalidLimit",
			input:   feed.GetFeedInput{Limit: 0},
			wantErr: item.ErrInvalidLimit,
		},
		{
			name:    "negative limit → ErrInvalidLimit",
			input:   feed.GetFeedInput{Limit: -10},
			wantErr: item.ErrInvalidLimit,
		},
		{
			name:    "unknown category → ErrInvalidCategory",
			input:   feed.GetFeedInput{Category: strPtr("watches"), Limit: 10},
			wantErr: item.ErrInvalidCategory,
		},
		{
			name:    "non-existent cursor → ErrInvalidCursor",
			input:   feed.GetFeedInput{Cursor: strPtr("does-not-exist"), Limit: 10},
			wantErr: item.ErrInvalidCursor,
		},
		{
			name: "cursor filtered out by seen_ids → ErrInvalidCursor",
			input: feed.GetFeedInput{
				Category: strPtr("cars"),
				SeenIDs:  []string{"car-01", "car-02"},
				Cursor:   strPtr("car-01"), // excluded before cursor lookup
				Limit:    10,
			},
			wantErr: item.ErrInvalidCursor,
		},
		{
			name: "cursor from wrong category → ErrInvalidCursor",
			input: feed.GetFeedInput{
				Category: strPtr("cars"),
				Cursor:   strPtr("house-01"), // not in cars pool
				Limit:    10,
			},
			wantErr: item.ErrInvalidCursor,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := svc.GetFeed(ctx(), tc.input)
			if err != tc.wantErr {
				t.Errorf("expected error %v, got %v", tc.wantErr, err)
			}
		})
	}
}

// ── Seen IDs exclusion ────────────────────────────────────────────────────────

func TestFeedService_SeenIDsExclusion(t *testing.T) {
	svc := newSvc()

	tests := []struct {
		name       string
		input      feed.GetFeedInput
		wantCount  int
		excludeIDs []string
	}{
		{
			name: "excludes 2 cars from pool of 7",
			input: feed.GetFeedInput{
				Category: strPtr("cars"),
				SeenIDs:  []string{"car-01", "car-02"},
				Limit:    10,
			},
			wantCount:  5, // 7 − 2
			excludeIDs: []string{"car-01", "car-02"},
		},
		{
			name: "excludes 3 cars from pool of 7",
			input: feed.GetFeedInput{
				Category: strPtr("cars"),
				SeenIDs:  []string{"car-01", "car-02", "car-03"},
				Limit:    10,
			},
			wantCount:  4, // 7 − 3
			excludeIDs: []string{"car-01", "car-02", "car-03"},
		},
		{
			name: "seen_ids order does not matter (same pool)",
			input: feed.GetFeedInput{
				Category: strPtr("cars"),
				SeenIDs:  []string{"car-03", "car-01", "car-02"}, // reversed
				Limit:    10,
			},
			wantCount:  4,
			excludeIDs: []string{"car-01", "car-02", "car-03"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			out, err := svc.GetFeed(ctx(), tc.input)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(out.Items) != tc.wantCount {
				t.Errorf("expected %d items, got %d", tc.wantCount, len(out.Items))
			}
			excluded := make(map[string]bool, len(tc.excludeIDs))
			for _, id := range tc.excludeIDs {
				excluded[id] = true
			}
			for _, it := range out.Items {
				if excluded[it.ID] {
					t.Errorf("item %q should have been excluded by seen_ids", it.ID)
				}
			}
		})
	}
}

// ── Empty result when all items are seen ──────────────────────────────────────

func TestFeedService_EmptyResult_WhenAllItemsSeen(t *testing.T) {
	svc := newSvc()

	tests := []struct {
		name  string
		input feed.GetFeedInput
	}{
		{
			name: "all 7 cars seen → empty result",
			input: feed.GetFeedInput{
				Category: strPtr("cars"),
				SeenIDs:  []string{"car-01", "car-02", "car-03", "car-04", "car-05", "car-06", "car-07"},
				Limit:    10,
			},
		},
		{
			name: "all 5 houses seen → empty result",
			input: feed.GetFeedInput{
				Category: strPtr("houses"),
				SeenIDs:  []string{"house-01", "house-02", "house-03", "house-04", "house-05"},
				Limit:    10,
			},
		},
		{
			name: "valid category with zero items (jewelry) → empty result",
			input: feed.GetFeedInput{
				Category: strPtr("jewelry"),
				Limit:    10,
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			out, err := svc.GetFeed(ctx(), tc.input)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(out.Items) != 0 {
				t.Errorf("expected 0 items, got %d", len(out.Items))
			}
			if out.NextCursor != nil {
				t.Errorf("expected nil next_cursor when pool is empty, got %q", *out.NextCursor)
			}
		})
	}
}

// ── Deterministic shuffle ─────────────────────────────────────────────────────

func TestFeedService_DeterministicShuffle_SameInputSameOrder(t *testing.T) {
	svc := newSvc()

	tests := []struct {
		name  string
		input feed.GetFeedInput
	}{
		{
			name:  "no category, no seen_ids",
			input: feed.GetFeedInput{Limit: 12},
		},
		{
			name:  "cars, one seen_id",
			input: feed.GetFeedInput{Category: strPtr("cars"), SeenIDs: []string{"car-01"}, Limit: 10},
		},
		{
			name:  "houses, two seen_ids",
			input: feed.GetFeedInput{Category: strPtr("houses"), SeenIDs: []string{"house-02", "house-04"}, Limit: 10},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			out1, err1 := svc.GetFeed(ctx(), tc.input)
			out2, err2 := svc.GetFeed(ctx(), tc.input)
			if err1 != nil || err2 != nil {
				t.Fatalf("unexpected errors: %v / %v", err1, err2)
			}
			if len(out1.Items) != len(out2.Items) {
				t.Fatalf("identical requests returned different counts: %d vs %d",
					len(out1.Items), len(out2.Items))
			}
			for i := range out1.Items {
				if out1.Items[i].ID != out2.Items[i].ID {
					t.Errorf("position %d: first call got %q, second got %q (non-deterministic)",
						i, out1.Items[i].ID, out2.Items[i].ID)
				}
			}
		})
	}
}

// ── Different seen_ids → different order ──────────────────────────────────────

func TestFeedService_DifferentSeenIDs_DifferentOrder(t *testing.T) {
	svc := newSvc()

	// Remove one different car each time; the common pool shares 5 cars whose
	// relative positions should differ because the shuffle seed changes.
	out1, err := svc.GetFeed(ctx(), feed.GetFeedInput{
		Category: strPtr("cars"),
		SeenIDs:  []string{"car-07"},
		Limit:    6,
	})
	if err != nil {
		t.Fatalf("req1 error: %v", err)
	}
	out2, err := svc.GetFeed(ctx(), feed.GetFeedInput{
		Category: strPtr("cars"),
		SeenIDs:  []string{"car-06"},
		Limit:    6,
	})
	if err != nil {
		t.Fatalf("req2 error: %v", err)
	}

	// Find common items (appear in both results).
	set2 := make(map[string]bool, len(out2.Items))
	for _, it := range out2.Items {
		set2[it.ID] = true
	}
	var common1 []string
	for _, it := range out1.Items {
		if set2[it.ID] {
			common1 = append(common1, it.ID)
		}
	}

	set1 := make(map[string]bool, len(out1.Items))
	for _, it := range out1.Items {
		set1[it.ID] = true
	}
	var common2 []string
	for _, it := range out2.Items {
		if set1[it.ID] {
			common2 = append(common2, it.ID)
		}
	}

	if len(common1) < 2 {
		t.Skip("not enough common items to compare ordering")
	}

	sameOrder := len(common1) == len(common2)
	if sameOrder {
		for i := range common1 {
			if common1[i] != common2[i] {
				sameOrder = false
				break
			}
		}
	}
	if sameOrder {
		t.Error("expected different ordering for different seen_ids, got identical order")
	}
}

// ── Pagination correctness ────────────────────────────────────────────────────

func TestFeedService_Pagination(t *testing.T) {
	svc := newSvc()

	tests := []struct {
		name      string
		input     feed.GetFeedInput
		wantCount int
		wantMore  bool // whether we expect a non-nil next_cursor
	}{
		{
			name:      "limit smaller than pool → returns limit items",
			input:     feed.GetFeedInput{Category: strPtr("cars"), Limit: 3},
			wantCount: 3,
			wantMore:  true,
		},
		{
			name:      "limit equal to pool → returns all, no next_cursor",
			input:     feed.GetFeedInput{Category: strPtr("cars"), Limit: 7},
			wantCount: 7,
			wantMore:  false,
		},
		{
			name:      "limit greater than pool → clamped to pool size",
			input:     feed.GetFeedInput{Category: strPtr("cars"), Limit: 50},
			wantCount: 7,
			wantMore:  false,
		},
		{
			name:      "limit > maxLimit (100) is clamped to 50",
			input:     feed.GetFeedInput{Limit: 100},
			wantCount: 12, // all 12 items seeded
			wantMore:  false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			out, err := svc.GetFeed(ctx(), tc.input)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(out.Items) != tc.wantCount {
				t.Errorf("expected %d items, got %d", tc.wantCount, len(out.Items))
			}
			if tc.wantMore && out.NextCursor == nil {
				t.Error("expected a next_cursor but got nil")
			}
			if !tc.wantMore && out.NextCursor != nil {
				t.Errorf("expected nil next_cursor, got %q", *out.NextCursor)
			}
		})
	}
}

// ── Next cursor correctness ───────────────────────────────────────────────────

func TestFeedService_NextCursorCorrectness(t *testing.T) {
	svc := newSvc()

	// Page 1 of cars with limit=2.
	page1, err := svc.GetFeed(ctx(), feed.GetFeedInput{
		Category: strPtr("cars"),
		Limit:    2,
	})
	if err != nil {
		t.Fatalf("page 1 error: %v", err)
	}
	if len(page1.Items) != 2 {
		t.Fatalf("expected 2 items on page 1, got %d", len(page1.Items))
	}
	if page1.NextCursor == nil {
		t.Fatal("expected next_cursor on page 1")
	}

	// next_cursor must equal the last item on page 1.
	lastIDPage1 := page1.Items[len(page1.Items)-1].ID
	if *page1.NextCursor != lastIDPage1 {
		t.Errorf("next_cursor %q does not match last item ID %q", *page1.NextCursor, lastIDPage1)
	}

	// Page 2 using the cursor — must not overlap with page 1.
	page2, err := svc.GetFeed(ctx(), feed.GetFeedInput{
		Category: strPtr("cars"),
		Cursor:   page1.NextCursor,
		Limit:    2,
	})
	if err != nil {
		t.Fatalf("page 2 error: %v", err)
	}
	if len(page2.Items) == 0 {
		t.Fatal("expected items on page 2, got none")
	}

	// No overlap.
	page1IDs := make(map[string]bool, len(page1.Items))
	for _, it := range page1.Items {
		page1IDs[it.ID] = true
	}
	for _, it := range page2.Items {
		if page1IDs[it.ID] {
			t.Errorf("item %q appeared on both pages", it.ID)
		}
	}
}

// ── Three-page full traversal (no duplicates) ─────────────────────────────────

func TestFeedService_FullTraversal_NoDuplicates(t *testing.T) {
	svc := newSvc()

	seen := make(map[string]bool)
	var cursor *string
	totalItems := 0
	pageLimit := 4

	for page := 1; page <= 4; page++ {
		out, err := svc.GetFeed(ctx(), feed.GetFeedInput{
			Limit:  pageLimit,
			Cursor: cursor,
		})
		if err != nil {
			t.Fatalf("page %d error: %v", page, err)
		}
		for _, it := range out.Items {
			if seen[it.ID] {
				t.Errorf("duplicate item %q on page %d", it.ID, page)
			}
			seen[it.ID] = true
		}
		totalItems += len(out.Items)
		cursor = out.NextCursor
		if cursor == nil {
			break
		}
	}

	// 12 items total in the repo; all must be visited.
	if totalItems != 12 {
		t.Errorf("expected 12 total items across pages, got %d", totalItems)
	}
}
