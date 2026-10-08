package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nexaCampus/backend-school-go/internal/cache"
	"github.com/nexaCampus/backend-school-go/internal/models"
)

type HallPassHandler struct {
	pool  *pgxpool.Pool
	cache cache.Cache
}

func NewHallPassHandler(pool *pgxpool.Pool, c ...cache.Cache) *HallPassHandler {
	var cc cache.Cache
	if len(c) > 0 && c[0] != nil {
		cc = c[0]
	} else {
		cc = cache.GetDefaultCache()
	}
	return &HallPassHandler{pool: pool, cache: cc}
}

// RequestPass submits a temporary hall pass request to the period teacher.
// Route: POST /v1/hallpass/request
func (h *HallPassHandler) RequestPass(w http.ResponseWriter, r *http.Request) {
	var req models.HallPassRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	req.StudentID = strings.TrimSpace(req.StudentID)
	if req.StudentID == "" {
		req.StudentID = resolveStudentID(r)
	}

	if req.StudentID == "" || req.Destination == "" {
		respondError(w, http.StatusBadRequest, "student_id and destination are required")
		return
	}

	if req.DurationMinutes <= 0 {
		req.DurationMinutes = 10
	}

	passID := fmt.Sprintf("hp-%d", time.Now().UnixNano())
	now := time.Now()
	expiresAt := now.Add(time.Duration(req.DurationMinutes) * time.Minute)
	teacherName := "Dr. Aris Thorne" // Active period teacher
	qrCode := fmt.Sprintf("HALLPASS|%s|%s|EXP:%d", passID, req.Destination, expiresAt.Unix())

	query := `
		INSERT INTO hall_passes (id, student_id, destination, duration_minutes, teacher_name, status, verification_qr, issued_at, expires_at)
		VALUES ($1, $2, $3, $4, $5, 'ACTIVE', $6, $7, $8)
		RETURNING id, student_id, destination, duration_minutes, teacher_name, status, verification_qr, issued_at, expires_at;
	`

	var hp models.HallPass
	err := h.pool.QueryRow(r.Context(), query,
		passID, req.StudentID, req.Destination, req.DurationMinutes, teacherName, qrCode, now, expiresAt,
	).Scan(
		&hp.ID, &hp.StudentID, &hp.Destination, &hp.DurationMinutes,
		&hp.TeacherName, &hp.Status, &hp.VerificationQR, &hp.IssuedAt, &hp.ExpiresAt,
	)

	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to issue hall pass")
		return
	}

	hp.RemainingSeconds = int(time.Until(expiresAt).Seconds())

	// Invalidate active pass cache for student
	if h.cache != nil {
		h.cache.Delete(fmt.Sprintf("hallpass:active:%s", req.StudentID))
	}

	respondJSON(w, http.StatusCreated, hp)
}

// GetActivePass retrieves the student's currently active digital hall pass.
// Route: GET /v1/hallpass/active?student_id={id}
func (h *HallPassHandler) GetActivePass(w http.ResponseWriter, r *http.Request) {
	studentID := resolveStudentID(r)
	if studentID == "" {
		respondError(w, http.StatusBadRequest, "student_id is required")
		return
	}

	// Check local cache
	passKey := fmt.Sprintf("hallpass:active:%s", studentID)
	if h.cache != nil {
		if val, found := h.cache.Get(passKey); found && val != nil {
			if cachedMap, ok := val.(map[string]interface{}); ok {
				respondJSON(w, http.StatusOK, cachedMap)
				return
			}
		}
	}

	query := `
		SELECT id, student_id, destination, duration_minutes, teacher_name, status, verification_qr, issued_at, expires_at
		FROM hall_passes
		WHERE student_id = $1 AND status = 'ACTIVE' AND expires_at > now()
		ORDER BY issued_at DESC
		LIMIT 1;
	`

	var hp models.HallPass
	err := h.pool.QueryRow(r.Context(), query, studentID).Scan(
		&hp.ID, &hp.StudentID, &hp.Destination, &hp.DurationMinutes,
		&hp.TeacherName, &hp.Status, &hp.VerificationQR, &hp.IssuedAt, &hp.ExpiresAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			// No active pass
			res := map[string]interface{}{
				"active":  false,
				"message": "No active hall pass at this time",
			}
			if h.cache != nil {
				h.cache.Set(passKey, res, 15*time.Second)
			}
			respondJSON(w, http.StatusOK, res)
			return
		}
		respondError(w, http.StatusInternalServerError, "Failed to query active hall pass")
		return
	}

	hp.RemainingSeconds = int(time.Until(hp.ExpiresAt).Seconds())
	if hp.RemainingSeconds < 0 {
		hp.RemainingSeconds = 0
		hp.Status = "EXPIRED"
	}

	res := map[string]interface{}{
		"active": true,
		"pass":   hp,
	}

	if h.cache != nil {
		h.cache.Set(passKey, res, 15*time.Second)
	}

	respondJSON(w, http.StatusOK, res)
}
