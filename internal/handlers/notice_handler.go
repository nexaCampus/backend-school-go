package handlers

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nexaCampus/backend-school-go/internal/cache"
	"github.com/nexaCampus/backend-school-go/internal/models"
)

type NoticeHandler struct {
	pool  *pgxpool.Pool
	cache cache.Cache
}

func NewNoticeHandler(pool *pgxpool.Pool, c ...cache.Cache) *NoticeHandler {
	var cc cache.Cache
	if len(c) > 0 && c[0] != nil {
		cc = c[0]
	} else {
		cc = cache.GetDefaultCache()
	}
	return &NoticeHandler{pool: pool, cache: cc}
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

	// 1. Check local cache before database query (Cache Hit)
	cacheKey := fmt.Sprintf("notices:list:%s:%d", strings.ToLower(category), limit)
	if h.cache != nil {
		if val, found := h.cache.Get(cacheKey); found && val != nil {
			if cachedNotices, ok := val.([]models.Notice); ok {
				respondJSON(w, http.StatusOK, cachedNotices)
				return
			}
		}
	}

	var query string
	var args []interface{}

	if category != "" && strings.ToLower(category) != "all" {
		query = `
			SELECT id, category, title, body, date_label, author, is_urgent, attachment_name, attachment_url, published_at
			FROM notices
			WHERE LOWER(category) = LOWER($1)
			ORDER BY is_urgent DESC, published_at DESC
			LIMIT $2;
		`
		args = []interface{}{category, limit}
	} else {
		query = `
			SELECT id, category, title, body, date_label, author, is_urgent, attachment_name, attachment_url, published_at
			FROM notices
			ORDER BY is_urgent DESC, published_at DESC
			LIMIT $1;
		`
		args = []interface{}{limit}
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

	// Cache circulars for 5 minutes
	if h.cache != nil {
		h.cache.Set(cacheKey, notices, 5*time.Minute)
	}

	respondJSON(w, http.StatusOK, notices)
}
