package health

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

// Handler holds the health HTTP handler.
type Handler struct {
	svc Service
}

// NewHandler returns a new health Handler with the given Service.
func NewHandler(svc Service) *Handler {
	return &Handler{svc: svc}
}

// Health handles GET /health and returns 200 OK.
func (h *Handler) Health(c echo.Context) error {
	if err := h.svc.Health(); err != nil {
		return err
	}
	return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
}
