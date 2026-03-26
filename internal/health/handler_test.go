package health_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"luxe-backend/internal/health"

	"github.com/labstack/echo/v4"
)

func TestHealthHandler_Health_Returns200(t *testing.T) {
	e := echo.New()
	svc := health.NewService()
	h := health.NewHandler(svc)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := h.Health(c)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}

	if rec.Body.String() != `{"status":"ok"}`+"\n" {
		t.Errorf("unexpected body: %s", rec.Body.String())
	}
}
