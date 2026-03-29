// Package itemhandler is the HTTP transport layer for the items endpoint.
// It sits above the feed layer in the dependency graph:
//
//	itemhandler → feed → item
//
// The handler parses HTTP requests, delegates to feed.Service for feed
// generation, and serialises the result to JSON.  It contains no business
// logic of its own.
package itemhandler

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/labstack/echo/v4"

	"luxe-backend/internal/feed"
	"luxe-backend/internal/shared"
)

// Handler holds HTTP handlers for the item resource.
type Handler struct {
	svc feed.Service
}

// NewHandler returns a new Handler backed by the given feed.Service.
func NewHandler(svc feed.Service) *Handler {
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

	// ── Delegate to feed service ─────────────────────────────────────────────
	out, err := h.svc.GetFeed(c.Request().Context(), feed.GetFeedInput{
		Category: category,
		Cursor:   cursor,
		Limit:    limit,
		SeenIDs:  seenIDs,
	})
	if err != nil {
		// feed re-exports the domain sentinel errors; compare against those.
		switch err {
		case feed.ErrInvalidCategory, feed.ErrInvalidCursor, feed.ErrInvalidLimit:
			return shared.JSON(c, http.StatusBadRequest, map[string]string{"error": err.Error()})
		default:
			return shared.JSON(c, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		}
	}

	// ── Build response ───────────────────────────────────────────────────────
	// out.Items is []item.Item; field access does not require importing item.
	resp := make([]itemResponse, 0, len(out.Items))
	for _, it := range out.Items {
		resp = append(resp, itemResponse{
			ID:       it.ID,
			Title:    it.Title,
			Price:    it.Price,
			ImageURL: it.ImageURL,
		})
	}

	return shared.JSON(c, http.StatusOK, listResponse{
		Items:      resp,
		NextCursor: out.NextCursor,
	})
}
