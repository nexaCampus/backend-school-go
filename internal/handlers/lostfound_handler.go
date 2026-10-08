package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nexaCampus/backend-school-go/internal/cache"
	"github.com/nexaCampus/backend-school-go/internal/models"
	"github.com/nexaCampus/backend-school-go/internal/security"
)

type LostFoundHandler struct {
	pool  *pgxpool.Pool
	cache cache.Cache
}

func NewLostFoundHandler(pool *pgxpool.Pool, c ...cache.Cache) *LostFoundHandler {
	var cc cache.Cache
	if len(c) > 0 && c[0] != nil {
		cc = c[0]
	} else {
		cc = cache.GetDefaultCache()
	}
	return &LostFoundHandler{pool: pool, cache: cc}
}

// ListItems returns reported lost and found articles.
// Route: GET /v1/lost-found/items?status={found|claimed}
func (h *LostFoundHandler) ListItems(w http.ResponseWriter, r *http.Request) {
	status := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("status")))

	// 1. Check local cache before database query (Cache Hit)
	cacheKey := fmt.Sprintf("lostfound:items:%s", status)
	if h.cache != nil {
		if val, found := h.cache.Get(cacheKey); found && val != nil {
			if cachedItems, ok := val.([]models.LostFoundItem); ok {
				respondJSON(w, http.StatusOK, cachedItems)
				return
			}
		}
	}

	var query string
	var args []interface{}

	if status != "" {
		query = `
			SELECT id, student_id, item_name, last_seen_location, image_url, status, created_at
			FROM lost_found_items
			WHERE LOWER(status) = LOWER($1)
			ORDER BY created_at DESC;
		`
		args = []interface{}{status}
	} else {
		query = `
			SELECT id, student_id, item_name, last_seen_location, image_url, status, created_at
			FROM lost_found_items
			ORDER BY created_at DESC;
		`
	}

	rows, err := h.pool.Query(r.Context(), query, args...)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to load lost and found items")
		return
	}
	defer rows.Close()

	items := make([]models.LostFoundItem, 0)
	for rows.Next() {
		var it models.LostFoundItem
		if err := rows.Scan(&it.ID, &it.StudentID, &it.ItemName, &it.LastSeenLocation, &it.ImageURL, &it.Status, &it.CreatedAt); err == nil {
			items = append(items, it)
		}
	}

	if h.cache != nil {
		h.cache.Set(cacheKey, items, 2*time.Minute)
	}

	respondJSON(w, http.StatusOK, items)
}

// ReportItem submits a missing article report.
// Route: POST /v1/lost-found/report
func (h *LostFoundHandler) ReportItem(w http.ResponseWriter, r *http.Request) {
	var req models.ReportLostItemRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	req.StudentID = strings.TrimSpace(req.StudentID)
	if req.StudentID == "" {
		req.StudentID = resolveStudentID(r)
	}

	if req.ItemName == "" || req.LastSeenLocation == "" {
		respondError(w, http.StatusBadRequest, "item_name and last_seen_location are required")
		return
	}

	if req.StudentID != "" && !VerifyStudentOwnership(r, req.StudentID) {
		respondError(w, http.StatusForbidden, "Forbidden: IDOR violation - cannot report lost item for another student")
		return
	}

	// Validate image URL for SSRF protection
	var img *string
	if strings.TrimSpace(req.ImageURL) != "" {
		trimmed := strings.TrimSpace(req.ImageURL)
		if err := security.ValidateSafeURL(trimmed); err != nil {
			respondError(w, http.StatusBadRequest, "Invalid image_url: "+err.Error())
			return
		}
		img = &trimmed
	}

	req.ItemName = security.SanitizeText(req.ItemName)
	req.LastSeenLocation = security.SanitizeText(req.LastSeenLocation)

	itemID := fmt.Sprintf("lf-%d", time.Now().UnixNano())

	query := `
		INSERT INTO lost_found_items (id, student_id, item_name, last_seen_location, image_url, status, created_at)
		VALUES ($1, $2, $3, $4, $5, 'reported', now())
		RETURNING id, student_id, item_name, last_seen_location, image_url, status, created_at;
	`

	var it models.LostFoundItem
	err := h.pool.QueryRow(r.Context(), query,
		itemID, req.StudentID, req.ItemName, req.LastSeenLocation, img,
	).Scan(&it.ID, &it.StudentID, &it.ItemName, &it.LastSeenLocation, &it.ImageURL, &it.Status, &it.CreatedAt)

	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to submit lost item report")
		return
	}

	// Invalidate lost & found cache
	if h.cache != nil {
		h.cache.InvalidatePrefix("lostfound:")
	}

	respondJSON(w, http.StatusCreated, it)
}
