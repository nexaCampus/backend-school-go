package handlers

import (
	"fmt"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nexaCampus/backend-school-go/internal/cache"
	"github.com/nexaCampus/backend-school-go/internal/models"
)

type FeeHandler struct {
	pool  *pgxpool.Pool
	cache cache.Cache
}

func NewFeeHandler(pool *pgxpool.Pool, c ...cache.Cache) *FeeHandler {
	var cc cache.Cache
	if len(c) > 0 && c[0] != nil {
		cc = c[0]
	} else {
		cc = cache.GetDefaultCache()
	}
	return &FeeHandler{pool: pool, cache: cc}
}

// GetFees computes fee breakdown, pending dues, and settled payment history.
// Route: GET /v1/fees?student_id={id}
func (h *FeeHandler) GetFees(w http.ResponseWriter, r *http.Request) {
	studentID := resolveStudentID(r)
	if studentID == "" {
		if r.URL.Query().Get("student_id") != "" {
			respondError(w, http.StatusForbidden, "Forbidden: IDOR violation - cannot access another student's fees")
			return
		}
		respondError(w, http.StatusBadRequest, "student_id is required")
		return
	}

	// 1. Check local cache before database query (Cache Hit)
	cacheKey := fmt.Sprintf("fees:summary:%s", studentID)
	if h.cache != nil {
		if val, found := h.cache.Get(cacheKey); found && val != nil {
			if cachedSummary, ok := val.(models.FeeSummary); ok {
				respondJSON(w, http.StatusOK, cachedSummary)
				return
			}
		}
	}

	// 1. Query fee items
	itemRows, err := h.pool.Query(r.Context(), `
		SELECT id, student_id, title, subtitle, amount, due_date, checked, cleared, cleared_meta
		FROM fee_items
		WHERE student_id = $1
		ORDER BY cleared ASC, id ASC;
	`, studentID)

	items := make([]models.FeeItem, 0)
	var totalDue int
	if err == nil {
		defer itemRows.Close()
		for itemRows.Next() {
			var it models.FeeItem
			if err := itemRows.Scan(
				&it.ID, &it.StudentID, &it.Title, &it.Subtitle, &it.Amount,
				&it.DueDate, &it.Checked, &it.Cleared, &it.ClearedMeta,
			); err == nil {
				if !it.Cleared {
					totalDue += it.Amount
				}
				items = append(items, it)
			}
		}
	}

	// 2. Query fee payment history
	histRows, err := h.pool.Query(r.Context(), `
		SELECT id, student_id, title, payment_date, amount, payment_ref
		FROM fee_history
		WHERE student_id = $1
		ORDER BY id DESC;
	`, studentID)

	history := make([]models.FeeHistory, 0)
	var totalPaid int
	if err == nil {
		defer histRows.Close()
		for histRows.Next() {
			var fh models.FeeHistory
			if err := histRows.Scan(
				&fh.ID, &fh.StudentID, &fh.Title, &fh.Date, &fh.Amount, &fh.Ref,
			); err == nil {
				totalPaid += fh.Amount
				history = append(history, fh)
			}
		}
	}

	summary := models.FeeSummary{
		StudentID: studentID,
		TotalDue:  totalDue,
		TotalPaid: totalPaid,
		DueDate:   "Oct 30, 2026",
		Currency:  "INR",
		Items:     items,
		History:   history,
	}

	// Cache fee summary for 5 minutes
	if h.cache != nil {
		h.cache.Set(cacheKey, summary, 5*time.Minute)
	}

	respondJSON(w, http.StatusOK, summary)
}
