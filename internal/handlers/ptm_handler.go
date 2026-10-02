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

type PtmHandler struct {
	pool *pgxpool.Pool
}

func NewPtmHandler(pool *pgxpool.Pool) *PtmHandler {
	return &PtmHandler{pool: pool}
}

// ListSessions returns upcoming parent-teacher meeting session dates.
// Route: GET /v1/ptm/sessions?grade={grade}
func (h *PtmHandler) ListSessions(w http.ResponseWriter, r *http.Request) {
	grade := strings.TrimSpace(r.URL.Query().Get("grade"))
	if grade == "" {
		grade = "12"
	}

	query := `
		SELECT id, grade, title, date, time_range, faculty_name, location
		FROM ptm_sessions
		WHERE grade = $1 OR grade = 'All'
		ORDER BY date ASC;
	`

	rows, err := h.pool.Query(r.Context(), query, grade)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to load PTM sessions")
		return
	}
	defer rows.Close()

	sessions := make([]models.PtmSession, 0)
	for rows.Next() {
		var s models.PtmSession
		if err := rows.Scan(&s.ID, &s.Grade, &s.Title, &s.Date, &s.TimeRange, &s.FacultyName, &s.Location); err == nil {
			sessions = append(sessions, s)
		}
	}

	respondJSON(w, http.StatusOK, sessions)
}

// ListSlots returns available conference time slots for a teacher and date.
// Route: GET /v1/ptm/slots?teacher_id={id}&date={date}
func (h *PtmHandler) ListSlots(w http.ResponseWriter, r *http.Request) {
	teacherID := strings.TrimSpace(r.URL.Query().Get("teacher_id"))
	date := strings.TrimSpace(r.URL.Query().Get("date"))

	var query string
	var args []interface{}

	if teacherID != "" && date != "" {
		query = `
			SELECT id, teacher_id, teacher_name, session_date, time_slot, is_booked
			FROM ptm_slots
			WHERE (teacher_id = $1 OR teacher_name ILIKE $1) AND session_date = $2
			ORDER BY time_slot ASC;
		`
		args = []interface{}{teacherID, date}
	} else if date != "" {
		query = `
			SELECT id, teacher_id, teacher_name, session_date, time_slot, is_booked
			FROM ptm_slots
			WHERE session_date = $1
			ORDER BY teacher_name ASC, time_slot ASC;
		`
		args = []interface{}{date}
	} else {
		query = `
			SELECT id, teacher_id, teacher_name, session_date, time_slot, is_booked
			FROM ptm_slots
			ORDER BY session_date ASC, time_slot ASC
			LIMIT 50;
		`
	}

	rows, err := h.pool.Query(r.Context(), query, args...)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to load PTM slots")
		return
	}
	defer rows.Close()

	slots := make([]models.PtmSlot, 0)
	for rows.Next() {
		var s models.PtmSlot
		if err := rows.Scan(&s.ID, &s.TeacherID, &s.TeacherName, &s.Date, &s.TimeSlot, &s.IsBooked); err == nil {
			slots = append(slots, s)
		}
	}

	respondJSON(w, http.StatusOK, slots)
}

// BookSlot reserves a meeting appointment slot.
// Route: POST /v1/ptm/book
func (h *PtmHandler) BookSlot(w http.ResponseWriter, r *http.Request) {
	var req models.PtmBookingRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	req.StudentID = strings.TrimSpace(req.StudentID)
	if req.StudentID == "" {
		req.StudentID = resolveStudentID(r)
	}

	if req.StudentID == "" || req.SlotID == "" {
		respondError(w, http.StatusBadRequest, "student_id and slot_id are required")
		return
	}

	// Fetch slot details
	var teacherName, sessionDate, timeSlot string
	var isBooked bool
	err := h.pool.QueryRow(r.Context(), `
		SELECT teacher_name, session_date, time_slot, is_booked
		FROM ptm_slots
		WHERE id = $1
		LIMIT 1;
	`, req.SlotID).Scan(&teacherName, &sessionDate, &timeSlot, &isBooked)

	if err != nil {
		respondError(w, http.StatusNotFound, "PTM appointment slot not found")
		return
	}

	if isBooked {
		respondError(w, http.StatusConflict, "Slot is already booked by another parent")
		return
	}

	bookingID := fmt.Sprintf("ptm-bk-%d", time.Now().UnixNano())
	meetingLink := fmt.Sprintf("https://meet.google.com/ptm-%s", req.SlotID)

	_, err = h.pool.Exec(r.Context(), `
		INSERT INTO ptm_bookings (id, student_id, slot_id, teacher_name, date, time_slot, agenda, meeting_link, status, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, 'CONFIRMED', now());
	`, bookingID, req.StudentID, req.SlotID, teacherName, sessionDate, timeSlot, req.Agenda, meetingLink)

	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to confirm PTM booking")
		return
	}

	// Mark slot as booked
	_, _ = h.pool.Exec(r.Context(), "UPDATE ptm_slots SET is_booked = true WHERE id = $1;", req.SlotID)

	respondJSON(w, http.StatusCreated, models.PtmBooking{
		ID:          bookingID,
		StudentID:   req.StudentID,
		SlotID:      req.SlotID,
		TeacherName: teacherName,
		Date:        sessionDate,
		TimeSlot:    timeSlot,
		Agenda:      req.Agenda,
		MeetingLink: meetingLink,
		Status:      "CONFIRMED",
		CreatedAt:   time.Now(),
	})
}
