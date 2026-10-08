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

type TimetableHandler struct {
	pool  *pgxpool.Pool
	cache cache.Cache
}

func NewTimetableHandler(pool *pgxpool.Pool, c ...cache.Cache) *TimetableHandler {
	var cc cache.Cache
	if len(c) > 0 && c[0] != nil {
		cc = c[0]
	} else {
		cc = cache.GetDefaultCache()
	}
	return &TimetableHandler{pool: pool, cache: cc}
}

// GetTimetable returns bell schedule and period timeline filtered by grade, section, and day of week.
// Route: GET /v1/timetable?grade={grade}&section={section}&day={1-6}
func (h *TimetableHandler) GetTimetable(w http.ResponseWriter, r *http.Request) {
	grade := strings.TrimSpace(r.URL.Query().Get("grade"))
	section := strings.TrimSpace(r.URL.Query().Get("section"))
	dayStr := strings.TrimSpace(r.URL.Query().Get("day"))

	if grade == "" {
		grade = "12"
	}
	if section == "" {
		section = "E"
	}

	// 1. Check local cache before database query (Cache Hit)
	cacheKey := fmt.Sprintf("timetable:%s:%s:%s", grade, section, dayStr)
	if h.cache != nil {
		if val, found := h.cache.Get(cacheKey); found && val != nil {
			if cachedSlots, ok := val.([]models.TimetableSlot); ok {
				respondJSON(w, http.StatusOK, cachedSlots)
				return
			}
		}
	}

	var query string
	var args []interface{}

	if dayStr != "" {
		day, err := strconv.Atoi(dayStr)
		if err == nil && day >= 1 && day <= 6 {
			query = `
				SELECT id, grade, section, day_of_week, period_label, time_label, subject, detail, room_number, teacher_name, is_live
				FROM timetable
				WHERE grade = $1 AND section = $2 AND day_of_week = $3
				ORDER BY id ASC;
			`
			args = []interface{}{grade, section, day}
		}
	}

	if query == "" {
		query = `
			SELECT id, grade, section, day_of_week, period_label, time_label, subject, detail, room_number, teacher_name, is_live
			FROM timetable
			WHERE grade = $1 AND section = $2
			ORDER BY day_of_week ASC, id ASC;
		`
		args = []interface{}{grade, section}
	}

	rows, err := h.pool.Query(r.Context(), query, args...)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to retrieve timetable")
		return
	}
	defer rows.Close()

	slots := make([]models.TimetableSlot, 0)
	for rows.Next() {
		var slot models.TimetableSlot
		if err := rows.Scan(
			&slot.ID, &slot.Grade, &slot.Section, &slot.DayOfWeek,
			&slot.PeriodLabel, &slot.Time, &slot.Subject, &slot.Detail,
			&slot.RoomNumber, &slot.Teacher, &slot.IsLive,
		); err == nil {
			slots = append(slots, slot)
		}
	}

	// Cache timetable slots for 15 minutes
	if h.cache != nil {
		h.cache.Set(cacheKey, slots, 15*time.Minute)
	}

	respondJSON(w, http.StatusOK, slots)
}
