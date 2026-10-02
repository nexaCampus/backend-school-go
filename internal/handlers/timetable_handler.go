package handlers

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nexaCampus/backend-school-go/internal/models"
)

type TimetableHandler struct {
	pool *pgxpool.Pool
}

func NewTimetableHandler(pool *pgxpool.Pool) *TimetableHandler {
	return &TimetableHandler{pool: pool}
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

	respondJSON(w, http.StatusOK, slots)
}
