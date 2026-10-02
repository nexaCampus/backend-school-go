package handlers

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nexaCampus/backend-school-go/internal/models"
)

type NoticeHandler struct {
	pool *pgxpool.Pool
}

func NewNoticeHandler(pool *pgxpool.Pool) *NoticeHandler {
	return &NoticeHandler{pool: pool}
}

// ListNotices retrieves official school circulars filtered by category and limit.
// Route: GET /v1/notices?category={all|urgent|academic|event}&limit={n}
func (h *NoticeHandler) ListNotices(w http.ResponseWriter, r *http.Request) {
	category := strings.TrimSpace(r.URL.Query().Get("category"))
	limitStr := strings.TrimSpace(r.URL.Query().Get("limit"))

	limit := 50
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 100 {
			limit = l
		}
	}

	var query string
	var args []interface{}

	if category != "" && strings.ToLower(category) != "all" {
		query = fmt.Sprintf(`
			SELECT id, category, title, body, date_label, author, is_urgent, attachment_name, attachment_url, published_at
			FROM notices
			WHERE LOWER(category) = LOWER($1)
			ORDER BY is_urgent DESC, published_at DESC
			LIMIT %d;
		`, limit)
		args = []interface{}{category}
	} else {
		query = fmt.Sprintf(`
			SELECT id, category, title, body, date_label, author, is_urgent, attachment_name, attachment_url, published_at
			FROM notices
			ORDER BY is_urgent DESC, published_at DESC
			LIMIT %d;
		`, limit)
	}

	rows, err := h.pool.Query(r.Context(), query, args...)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to fetch notices")
		return
	}
	defer rows.Close()

	notices := make([]models.Notice, 0)
	for rows.Next() {
		var n models.Notice
		if err := rows.Scan(
			&n.ID, &n.Category, &n.Title, &n.Body, &n.Date,
			&n.Author, &n.IsUrgent, &n.AttachmentName, &n.AttachmentURL, &n.PublishedAt,
		); err == nil {
			notices = append(notices, n)
		}
	}

	respondJSON(w, http.StatusOK, notices)
}
