package item_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"luxe-backend/internal/item"

	"github.com/labstack/echo/v4"
)

// itemsResponse mirrors the JSON shape returned by GET /api/v1/items.
type itemsResponse struct {
	Items      []map[string]interface{} `json:"items"`
	NextCursor *string                  `json:"next_cursor"`
}

// newTestHandler wires up a real service with an in-memory repository.
func newTestHandler() *item.Handler {
	repo := item.NewInMemoryRepository()
	svc := item.NewService(repo)
	return item.NewHandler(svc)
}

// getItems fires GET /api/v1/items with an optional raw query string (e.g. "category=cars&limit=5").
func getItems(t *testing.T, query string) *httptest.ResponseRecorder {
	t.Helper()
	e := echo.New()
	h := newTestHandler()

	url := "/api/v1/items"
	if query != "" {
		url += "?" + query
	}

	req := httptest.NewRequest(http.MethodGet, url, nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	if err := h.GetItems(c); err != nil {
		t.Fatalf("handler returned unexpected error: %v", err)
	}
	return rec
}

// parseItemsResponse unmarshals the recorder body into itemsResponse.
func parseItemsResponse(t *testing.T, rec *httptest.ResponseRecorder) itemsResponse {
	t.Helper()
	var resp itemsResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("invalid json: %v\nbody: %s", err, rec.Body.String())
	}
	return resp
}

// ─── Response shape ───────────────────────────────────────────────────────────

func TestGetItems_ResponseShape(t *testing.T) {
	rec := getItems(t, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	var raw map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &raw); err != nil {
		t.Fatalf("invalid json: %v", err)
	}
	if _, ok := raw["items"]; !ok {
		t.Error("response missing 'items' field")
	}
	if _, ok := raw["next_cursor"]; !ok {
		t.Error("response missing 'next_cursor' field")
	}
}

// ─── First page (no cursor) ───────────────────────────────────────────────────

func TestGetItems_FirstPage_ReturnsDefaultLimit(t *testing.T) {
	rec := getItems(t, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	resp := parseItemsResponse(t, rec)
	if len(resp.Items) != 10 {
		t.Errorf("expected 10 items (default limit), got %d", len(resp.Items))
	}
}

func TestGetItems_FirstPage_HasNextCursor(t *testing.T) {
	rec := getItems(t, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	resp := parseItemsResponse(t, rec)
	if resp.NextCursor == nil {
		t.Error("expected next_cursor to be set on first page when more items remain, got nil")
	}
}

// ─── Pagination ───────────────────────────────────────────────────────────────

func TestGetItems_Pagination_NextPage(t *testing.T) {
	// First page.
	rec1 := getItems(t, "limit=3")
	if rec1.Code != http.StatusOK {
		t.Fatalf("expected 200 on first page, got %d", rec1.Code)
	}
	resp1 := parseItemsResponse(t, rec1)
	if resp1.NextCursor == nil {
		t.Fatal("expected next_cursor on first page")
	}

	// Second page using the cursor.
	rec2 := getItems(t, fmt.Sprintf("limit=3&cursor=%s", *resp1.NextCursor))
	if rec2.Code != http.StatusOK {
		t.Fatalf("expected 200 on second page, got %d", rec2.Code)
	}
	resp2 := parseItemsResponse(t, rec2)
	if len(resp2.Items) == 0 {
		t.Fatal("expected items on second page, got none")
	}

	// Pages must not overlap.
	ids1 := make(map[string]bool)
	for _, it := range resp1.Items {
		ids1[it["id"].(string)] = true
	}
	for _, it := range resp2.Items {
		if ids1[it["id"].(string)] {
			t.Errorf("item %q appears on both page 1 and page 2", it["id"])
		}
	}
}

// ─── End of list ─────────────────────────────────────────────────────────────

func TestGetItems_EndOfList_NilNextCursor(t *testing.T) {
	// limit=50 should exhaust the seeded list, so next_cursor must be nil.
	rec := getItems(t, "limit=50")
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	resp := parseItemsResponse(t, rec)
	if resp.NextCursor != nil {
		t.Errorf("expected nil next_cursor at end of list, got %q", *resp.NextCursor)
	}
}

// ─── Invalid cursor ───────────────────────────────────────────────────────────

func TestGetItems_InvalidCursor_Returns400(t *testing.T) {
	rec := getItems(t, "cursor=nonexistent-id-xyz")
	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for invalid cursor, got %d", rec.Code)
	}
	var body map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("could not unmarshal error body: %v", err)
	}
	if _, ok := body["error"]; !ok {
		t.Error("expected 'error' key in 400 response body")
	}
}

// ─── Limit behaviour ─────────────────────────────────────────────────────────

func TestGetItems_Limit_MaxClamped50(t *testing.T) {
	// limit=100 must be silently clamped to 50; status must still be 200.
	rec := getItems(t, "limit=100")
	if rec.Code != http.StatusOK {
		t.Errorf("expected 200 when limit is over max, got %d", rec.Code)
	}
	resp := parseItemsResponse(t, rec)
	if len(resp.Items) > 50 {
		t.Errorf("clamped limit should return at most 50 items, got %d", len(resp.Items))
	}
}

func TestGetItems_Limit_ZeroReturns400(t *testing.T) {
	rec := getItems(t, "limit=0")
	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for limit=0, got %d", rec.Code)
	}
}

func TestGetItems_Limit_NegativeReturns400(t *testing.T) {
	rec := getItems(t, "limit=-5")
	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for limit=-5, got %d", rec.Code)
	}
}

// ─── Category + pagination ────────────────────────────────────────────────────

func TestGetItems_Category_WithPagination(t *testing.T) {
	// First page of cars with a small limit.
	rec1 := getItems(t, "category=cars&limit=2")
	if rec1.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec1.Code)
	}
	resp1 := parseItemsResponse(t, rec1)
	if len(resp1.Items) != 2 {
		t.Fatalf("expected 2 cars on first page, got %d", len(resp1.Items))
	}
	if resp1.NextCursor == nil {
		t.Fatal("expected next_cursor after first car page")
	}

	// Second page of cars.
	rec2 := getItems(t, fmt.Sprintf("category=cars&limit=2&cursor=%s", *resp1.NextCursor))
	if rec2.Code != http.StatusOK {
		t.Fatalf("expected 200 on second car page, got %d", rec2.Code)
	}
	resp2 := parseItemsResponse(t, rec2)
	if len(resp2.Items) == 0 {
		t.Fatal("expected items on second car page, got none")
	}

	// Items from page 2 must not overlap with page 1.
	ids1 := make(map[string]bool)
	for _, it := range resp1.Items {
		ids1[it["id"].(string)] = true
	}
	for _, it := range resp2.Items {
		if ids1[it["id"].(string)] {
			t.Errorf("car %q appeared on both pages", it["id"])
		}
	}
}

// ─── Existing tests (updated for new response shape) ─────────────────────────

func TestGetItems_ReturnsAllItems(t *testing.T) {
	rec := getItems(t, "")
	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
	resp := parseItemsResponse(t, rec)
	if len(resp.Items) < 5 {
		t.Errorf("expected >=5 items, got %d", len(resp.Items))
	}
	for _, it := range resp.Items {
		for _, field := range []string{"id", "title", "price", "image_url"} {
			if _, ok := it[field]; !ok {
				t.Errorf("item missing field %q", field)
			}
		}
	}
}

func TestGetItems_ReturnsCars(t *testing.T) {
	rec := getItems(t, "category=cars")
	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
	resp := parseItemsResponse(t, rec)
	if len(resp.Items) == 0 {
		t.Fatal("expected at least one car item, got none")
	}
}

func TestGetItems_ReturnsHouses(t *testing.T) {
	rec := getItems(t, "category=houses")
	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
	resp := parseItemsResponse(t, rec)
	if len(resp.Items) == 0 {
		t.Fatal("expected at least one house item, got none")
	}
}

func TestGetItems_Returns200_EmptyArray_WhenNoCategoryMatches(t *testing.T) {
	// "jewelry" is a valid category but has no seeded items → expect empty items[] with 200.
	rec := getItems(t, "category=jewelry")
	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
	resp := parseItemsResponse(t, rec)
	if len(resp.Items) != 0 {
		t.Errorf("expected empty items array, got %d items", len(resp.Items))
	}
}

func TestGetItems_CursorNotInFilteredSet_Returns400(t *testing.T) {
    rec := getItems(t, "category=cars&cursor=house-1")
    if rec.Code != http.StatusBadRequest {
        t.Errorf("expected 400, got %d", rec.Code)
    }
}

func TestGetItems_Returns400_WhenCategoryIsInvalid(t *testing.T) {
	rec := getItems(t, "category=watches")
	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}
}

func TestGetItems_Returns400_ResponseBody_ContainsError(t *testing.T) {
	rec := getItems(t, "category=invalid")
	var body map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("could not unmarshal response: %v", err)
	}
	if _, ok := body["error"]; !ok {
		t.Errorf("expected 'error' key in response body, got %v", body)
	}
}

// ─── Dream Mode: seen_ids ─────────────────────────────────────────────────────

// TestGetItems_SeenIDs_ExcludesSeenItems verifies that items whose IDs appear
// in the seen_ids param are absent from the response.
func TestGetItems_SeenIDs_ExcludesSeenItems(t *testing.T) {
	rec := getItems(t, "category=cars&seen_ids=car-01,car-02")
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	resp := parseItemsResponse(t, rec)
	seenSet := map[string]bool{"car-01": true, "car-02": true}
	for _, it := range resp.Items {
		id := it["id"].(string)
		if seenSet[id] {
			t.Errorf("item %q should have been excluded via seen_ids", id)
		}
	}
}

// TestGetItems_SeenIDs_AllItemsSeen_ReturnsEmptyList verifies that when every
// item in the category has been seen, the response is an empty list with a nil
// next_cursor.
func TestGetItems_SeenIDs_AllItemsSeen_ReturnsEmptyList(t *testing.T) {
	// All 7 cars are passed as seen_ids.
	seenIDs := "car-01,car-02,car-03,car-04,car-05,car-06,car-07"
	rec := getItems(t, "category=cars&seen_ids="+seenIDs)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	resp := parseItemsResponse(t, rec)
	if len(resp.Items) != 0 {
		t.Errorf("expected 0 items when all cars are seen, got %d", len(resp.Items))
	}
	if resp.NextCursor != nil {
		t.Errorf("expected nil next_cursor when pool is empty, got %q", *resp.NextCursor)
	}
}

// TestGetItems_Shuffle_Deterministic verifies that two identical requests always
// return items in the same order (deterministic seeded shuffle).
func TestGetItems_Shuffle_Deterministic(t *testing.T) {
	rec1 := getItems(t, "category=cars&seen_ids=car-01&limit=6")
	rec2 := getItems(t, "category=cars&seen_ids=car-01&limit=6")

	resp1 := parseItemsResponse(t, rec1)
	resp2 := parseItemsResponse(t, rec2)

	if len(resp1.Items) != len(resp2.Items) {
		t.Fatalf("identical requests returned different item counts: %d vs %d",
			len(resp1.Items), len(resp2.Items))
	}
	for i := range resp1.Items {
		id1 := resp1.Items[i]["id"].(string)
		id2 := resp2.Items[i]["id"].(string)
		if id1 != id2 {
			t.Errorf("position %d: first call got %q, second call got %q (non-deterministic shuffle)",
				i, id1, id2)
		}
	}
}

// TestGetItems_Shuffle_DifferentSeenIDs_DifferentOrder verifies that different
// seen_ids values produce different shuffle orderings of the remaining items.
func TestGetItems_Shuffle_DifferentSeenIDs_DifferentOrder(t *testing.T) {
	// Pool for req1: cars minus car-07 = [car-01..car-06]
	// Pool for req2: cars minus car-06 = [car-01..car-05, car-07]
	// Common items: car-01..car-05 should appear in different relative positions.
	rec1 := getItems(t, "category=cars&seen_ids=car-07&limit=5")
	rec2 := getItems(t, "category=cars&seen_ids=car-06&limit=5")

	resp1 := parseItemsResponse(t, rec1)
	resp2 := parseItemsResponse(t, rec2)

	if len(resp1.Items) == 0 || len(resp2.Items) == 0 {
		t.Skip("not enough items to compare")
	}

	// Build ordered lists of common items in each response.
	set2 := make(map[string]bool, len(resp2.Items))
	for _, it := range resp2.Items {
		set2[it["id"].(string)] = true
	}
	var common1 []string
	for _, it := range resp1.Items {
		if set2[it["id"].(string)] {
			common1 = append(common1, it["id"].(string))
		}
	}
	set1 := make(map[string]bool, len(resp1.Items))
	for _, it := range resp1.Items {
		set1[it["id"].(string)] = true
	}
	var common2 []string
	for _, it := range resp2.Items {
		if set1[it["id"].(string)] {
			common2 = append(common2, it["id"].(string))
		}
	}

	if len(common1) < 2 {
		t.Skip("not enough common items to compare order")
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
		t.Error("expected different item ordering for different seen_ids, but got identical order")
	}
}

// TestGetItems_Pagination_NoDuplicatesAfterShuffle verifies that seen_ids-based
// pagination produces no duplicates across pages even when the shuffle seed
// changes between pages.
func TestGetItems_Pagination_NoDuplicatesAfterShuffle(t *testing.T) {
	// Page 1: no seen_ids, limit=3.
	rec1 := getItems(t, "limit=3")
	if rec1.Code != http.StatusOK {
		t.Fatalf("page 1 expected 200, got %d", rec1.Code)
	}
	resp1 := parseItemsResponse(t, rec1)
	if len(resp1.Items) == 0 {
		t.Fatal("page 1 returned no items")
	}

	// Collect page-1 IDs for seen_ids param on page 2.
	ids1 := make([]string, 0, len(resp1.Items))
	for _, it := range resp1.Items {
		ids1 = append(ids1, it["id"].(string))
	}
	seenParam := strings.Join(ids1, ",")

	// Page 2: pass page-1 items as seen_ids (Dream Mode accumulation).
	rec2 := getItems(t, "limit=3&seen_ids="+seenParam)
	if rec2.Code != http.StatusOK {
		t.Fatalf("page 2 expected 200, got %d", rec2.Code)
	}
	resp2 := parseItemsResponse(t, rec2)

	// No item from page 1 must appear on page 2.
	seenSet := make(map[string]bool, len(ids1))
	for _, id := range ids1 {
		seenSet[id] = true
	}
	for _, it := range resp2.Items {
		id := it["id"].(string)
		if seenSet[id] {
			t.Errorf("item %q appeared on both pages (seen_ids pagination)", id)
		}
	}
}

// TestGetItems_CursorValidation_SeenIDsFilteredOut verifies that a cursor
// pointing to an item that was excluded by seen_ids returns 400.
func TestGetItems_CursorValidation_SeenIDsFilteredOut(t *testing.T) {
	// car-01 is in seen_ids, so it is removed from the pool before cursor lookup.
	rec := getItems(t, "category=cars&seen_ids=car-01,car-02&cursor=car-01")
	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400 when cursor is excluded by seen_ids, got %d", rec.Code)
	}
	var body map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("could not unmarshal error body: %v", err)
	}
	if _, ok := body["error"]; !ok {
		t.Error("expected 'error' key in 400 response body")
	}
}

// TestGetItems_CategoryAndSeenIDs_Combined verifies that category filtering and
// seen_ids exclusion work correctly together.
func TestGetItems_CategoryAndSeenIDs_Combined(t *testing.T) {
	// 7 cars seeded; exclude 3 → at most 4 cars should be returned.
	rec := getItems(t, "category=cars&seen_ids=car-01,car-02,car-03&limit=10")
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	resp := parseItemsResponse(t, rec)

	excluded := map[string]bool{"car-01": true, "car-02": true, "car-03": true}
	for _, it := range resp.Items {
		id := it["id"].(string)
		if excluded[id] {
			t.Errorf("item %q should have been excluded via seen_ids", id)
		}
	}
	// 7 cars total − 3 excluded = 4 remaining.
	if len(resp.Items) > 4 {
		t.Errorf("expected at most 4 items (7 cars − 3 seen), got %d", len(resp.Items))
	}
}

// TestGetItems_DreamMode_ResponseShapeUnchanged verifies that adding seen_ids
// does not alter the JSON response envelope shape.
func TestGetItems_DreamMode_ResponseShapeUnchanged(t *testing.T) {
	rec := getItems(t, "seen_ids=car-01")
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	var raw map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &raw); err != nil {
		t.Fatalf("invalid json: %v", err)
	}
	for _, field := range []string{"items", "next_cursor"} {
		if _, ok := raw[field]; !ok {
			t.Errorf("response missing top-level field %q", field)
		}
	}
	items, ok := raw["items"].([]interface{})
	if !ok {
		t.Fatal("'items' is not an array")
	}
	for _, rawItem := range items {
		obj, ok := rawItem.(map[string]interface{})
		if !ok {
			t.Fatal("item is not an object")
		}
		for _, f := range []string{"id", "title", "price", "image_url"} {
			if _, ok := obj[f]; !ok {
				t.Errorf("item missing field %q", f)
			}
		}
	}
}
