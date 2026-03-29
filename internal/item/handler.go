package item

import (
	"net/http"
	"strconv"
	"strings"

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

// listResponse is the top-level JSON envelope for GET /api/v1/items.
type listResponse struct {
	Items      []itemResponse `json:"items"`
	NextCursor *string        `json:"next_cursor"`
}

const defaultLimit = 10

// GetItems handles GET /api/v1/items.
//
// Query params:
//   - category (optional): cars | houses | jewelry
//   - cursor   (optional): ID of the last item seen on the previous page
//   - limit    (optional): page size, default 10, max 50
//   - seen_ids (optional): comma-separated item IDs to exclude (Dream Mode feed)
func (h *Handler) GetItems(c echo.Context) error {
	// ── Parse category ───────────────────────────────────────────────────────
	var category *string
	if raw := c.QueryParam("category"); raw != "" {
		cat := raw
		category = &cat
	}

	// ── Parse cursor ─────────────────────────────────────────────────────────
	var cursor *string
	if raw := c.QueryParam("cursor"); raw != "" {
		cur := raw
		cursor = &cur
	}

	// ── Parse limit ──────────────────────────────────────────────────────────
	limit := defaultLimit
	if raw := c.QueryParam("limit"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil {
			return shared.JSON(c, http.StatusBadRequest, map[string]string{
				"error": "limit must be a valid integer",
			})
		}
		limit = parsed
	}

	// ── Parse seen_ids ───────────────────────────────────────────────────────
	var seenIDs []string
	if raw := c.QueryParam("seen_ids"); raw != "" {
		parts := strings.Split(raw, ",")
		seenIDs = make([]string, 0, len(parts))
		for _, p := range parts {
			if trimmed := strings.TrimSpace(p); trimmed != "" {
				seenIDs = append(seenIDs, trimmed)
			}
		}
	}

	// ── Delegate to service ──────────────────────────────────────────────────
	items, nextCursor, err := h.svc.GetItems(c.Request().Context(), category, cursor, limit, seenIDs)
	if err != nil {
		switch err {
		case ErrInvalidCategory, ErrInvalidCursor, ErrInvalidLimit:
			return shared.JSON(c, http.StatusBadRequest, map[string]string{"error": err.Error()})
		default:
			return shared.JSON(c, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		}
	}

	// ── Build response ───────────────────────────────────────────────────────
	resp := make([]itemResponse, 0, len(items))
	for _, it := range items {
		resp = append(resp, itemResponse{
			ID:       it.ID,
			Title:    it.Title,
			Price:    it.Price,
			ImageURL: it.ImageURL,
		})
	}

	return shared.JSON(c, http.StatusOK, listResponse{
		Items:      resp,
		NextCursor: nextCursor,
	})
}
