package item_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
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
