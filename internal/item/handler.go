package item

import (
	"net/http"

	"github.com/labstack/echo/v4"

	"luxe-backend/internal/shared"
)

// Handler holds HTTP handlers for the item resource.
type Handler struct {
	svc Service
}

// NewHandler returns a new item Handler with the given Service.
func NewHandler(svc Service) *Handler {
	return &Handler{svc: svc}
}

// itemResponse is the JSON shape returned for each item.
type itemResponse struct {
	ID       string  `json:"id"`
	Title    string  `json:"title"`
	Price    float64 `json:"price"`
	ImageURL string  `json:"image_url"`
}

// GetItems handles GET /api/v1/items.
// Optional query param: category (cars | houses | jewelry).
func (h *Handler) GetItems(c echo.Context) error {
	var category *string
	if raw := c.QueryParam("category"); raw != "" {
		cat := raw
		category = &cat
	}

	items, err := h.svc.GetItems(c.Request().Context(), category)
	if err != nil {
		switch err {
		case ErrInvalidCategory:
			return shared.JSON(c, http.StatusBadRequest, map[string]string{"error": err.Error()})
		default:
			return shared.JSON(c, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		}
	}

	resp := make([]itemResponse, 0, len(items))
	for _, it := range items {
		resp = append(resp, itemResponse{
			ID:       it.ID,
			Title:    it.Title,
			Price:    it.Price,
			ImageURL: it.ImageURL,
		})
	}

	return shared.JSON(c, http.StatusOK, resp)
}
