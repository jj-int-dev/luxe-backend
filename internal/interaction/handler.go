package interaction

import (
	"net/http"

	"github.com/labstack/echo/v4"

	"luxe-backend/internal/shared"
)

// Handler holds HTTP handlers for the interaction resource.
type Handler struct {
	svc Service
}

// NewHandler returns a new interaction Handler with the given Service.
func NewHandler(svc Service) *Handler {
	return &Handler{svc: svc}
}

// createRequest is the expected JSON body for POST /api/v1/interactions.
type createRequest struct {
	ItemID string `json:"item_id"`
	Action string `json:"action"`
}

// Create handles POST /api/v1/interactions.
func (h *Handler) Create(c echo.Context) error {
	var req createRequest

	if err := c.Bind(&req); err != nil {
		return shared.JSON(c, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
	}
	if req.ItemID == "" || req.Action == "" {
		return shared.JSON(c, http.StatusBadRequest, map[string]string{"error": "missing required fields"})
	}

	err := h.svc.Create(c.Request().Context(), req.ItemID, req.Action)
	if err != nil {
		switch err {
		case ErrInvalidItemID, ErrInvalidAction:
			return shared.JSON(c, http.StatusBadRequest, map[string]string{"error": err.Error()})
		default:
			return shared.JSON(c, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		}
	}

	return shared.JSON(c, http.StatusOK, map[string]string{"status": "ok"})
}
