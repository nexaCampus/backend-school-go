package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nexaCampus/backend-school-go/internal/models"
)

type LostFoundHandler struct {
	pool *pgxpool.Pool
}

func NewLostFoundHandler(pool *pgxpool.Pool) *LostFoundHandler {
	return &LostFoundHandler{pool: pool}
}

// ListItems returns reported lost and found articles.
// Route: GET /v1/lost-found/items?status={found|claimed}
func (h *LostFoundHandler) ListItems(w http.ResponseWriter, r *http.Request) {
	status := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("status")))

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

	itemID := fmt.Sprintf("lf-%d", time.Now().UnixNano())
	var img *string
	if strings.TrimSpace(req.ImageURL) != "" {
		trimmed := strings.TrimSpace(req.ImageURL)
		img = &trimmed
	}

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

	respondJSON(w, http.StatusCreated, it)
}
