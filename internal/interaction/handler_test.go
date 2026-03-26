package interaction_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"luxe-backend/internal/interaction"

	"github.com/labstack/echo/v4"
)

// newTestHandler wires up a real service with an in-memory repository.
func newTestHandler() *interaction.Handler {
	repo := interaction.NewInMemoryRepository()
	svc := interaction.NewService(repo)
	return interaction.NewHandler(svc)
}

func postInteraction(t *testing.T, body string) *httptest.ResponseRecorder {
	t.Helper()
	e := echo.New()
	h := newTestHandler()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/interactions", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	if err := h.Create(c); err != nil {
		t.Fatalf("handler returned unexpected error: %v", err)
	}
	return rec
}

// --- Happy path ---

func TestCreate_Returns200_WhenActionIsLike(t *testing.T) {
	rec := postInteraction(t, `{"item_id":"item-1","action":"like"}`)
	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
}

func TestCreate_Returns200_WhenActionIsSkip(t *testing.T) {
	rec := postInteraction(t, `{"item_id":"item-1","action":"skip"}`)
	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
}

func TestCreate_ResponseBody_ContainsStatus(t *testing.T) {
	rec := postInteraction(t, `{"item_id":"item-1","action":"like"}`)
	want := `{"status":"ok"}` + "\n"
	if rec.Body.String() != want {
		t.Errorf("unexpected body: got %q, want %q", rec.Body.String(), want)
	}
}

// --- Validation: item_id ---

func TestCreate_Returns400_WhenItemIDMissing(t *testing.T) {
	rec := postInteraction(t, `{"action":"like"}`)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}
}

func TestCreate_Returns400_WhenItemIDEmpty(t *testing.T) {
	rec := postInteraction(t, `{"item_id":"","action":"like"}`)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}
}

// --- Validation: action ---

func TestCreate_Returns400_WhenActionMissing(t *testing.T) {
	rec := postInteraction(t, `{"item_id":"item-1"}`)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}
}

func TestCreate_Returns400_WhenActionInvalid(t *testing.T) {
	rec := postInteraction(t, `{"item_id":"item-1","action":"dislike"}`)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}
}

func TestCreate_Returns400_WhenActionIsRandomString(t *testing.T) {
	rec := postInteraction(t, `{"item_id":"item-1","action":"LIKE"}`)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d; action validation should be case-sensitive", rec.Code)
	}
}
