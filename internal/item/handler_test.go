package item_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"luxe-backend/internal/item"

	"github.com/labstack/echo/v4"
)

// newTestHandler wires up a real service with an in-memory repository.
func newTestHandler() *item.Handler {
	repo := item.NewInMemoryRepository()
	svc := item.NewService(repo)
	return item.NewHandler(svc)
}

// getItems fires GET /api/v1/items with an optional raw query string (e.g. "category=cars").
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

	// Echo parses query params from the request URL automatically.
	if err := h.GetItems(c); err != nil {
		t.Fatalf("handler returned unexpected error: %v", err)
	}
	return rec
}

// --- Happy path: no category filter ---

func TestGetItems_Returns200_WithAllItems(t *testing.T) {
	rec := getItems(t, "")
	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
}

func TestGetItems_ReturnsJSON_Array(t *testing.T) {
	rec := getItems(t, "")
	var items []map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &items); err != nil {
		t.Fatalf("response is not a JSON array: %v", err)
	}
}

func TestGetItems_ReturnsAllSeededItems(t *testing.T) {
	rec := getItems(t, "")
	var items []map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &items); err != nil {
		t.Fatalf("could not unmarshal response: %v", err)
	}
	if len(items) < 5 {
		t.Errorf("expected at least 5 items, got %d", len(items))
	}
}

func TestGetItems_EachItem_HasRequiredFields(t *testing.T) {
	rec := getItems(t, "")
	var items []map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &items); err != nil {
		t.Fatalf("could not unmarshal response: %v", err)
	}
	for _, it := range items {
		for _, field := range []string{"id", "title", "price", "image_url"} {
			if _, ok := it[field]; !ok {
				t.Errorf("item missing field %q: %v", field, it)
			}
		}
	}
}

// --- Happy path: valid category filter ---

func TestGetItems_Returns200_WhenCategoryIsCars(t *testing.T) {
	rec := getItems(t, "category=cars")
	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
}

func TestGetItems_FiltersByCars_ReturnsOnlyCars(t *testing.T) {
	rec := getItems(t, "category=cars")
	var items []map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &items); err != nil {
		t.Fatalf("could not unmarshal response: %v", err)
	}
	if len(items) == 0 {
		t.Fatal("expected at least one car item, got none")
	}
}

func TestGetItems_Returns200_WhenCategoryIsHouses(t *testing.T) {
	rec := getItems(t, "category=houses")
	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
}

// --- Happy path: valid category with no matching items ---

func TestGetItems_Returns200_EmptyArray_WhenNoCategoryMatches(t *testing.T) {
	// "jewelry" is a valid category but has no seeded items → expect [] with 200.
	rec := getItems(t, "category=jewelry")
	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
	var items []map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &items); err != nil {
		t.Fatalf("could not unmarshal response: %v", err)
	}
	if len(items) != 0 {
		t.Errorf("expected empty array, got %d items", len(items))
	}
}

// --- Invalid category → 400 ---

func TestGetItems_Returns400_WhenCategoryIsInvalid(t *testing.T) {
	rec := getItems(t, "category=watches")
	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}
}

func TestGetItems_Returns400_WhenCategoryIsRandomString(t *testing.T) {
	rec := getItems(t, "category=Cars") // case-sensitive
	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d; category validation should be case-sensitive", rec.Code)
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
