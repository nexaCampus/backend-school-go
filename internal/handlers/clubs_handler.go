package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nexaCampus/backend-school-go/internal/models"
)

type ClubsHandler struct {
	pool *pgxpool.Pool
}

func NewClubsHandler(pool *pgxpool.Pool) *ClubsHandler {
	return &ClubsHandler{pool: pool}
}

// ListEnrolled returns student's joined extracurricular clubs and upcoming meeting times.
// Route: GET /v1/clubs/enrolled?student_id={id}
func (h *ClubsHandler) ListEnrolled(w http.ResponseWriter, r *http.Request) {
	studentID := resolveStudentID(r)
	if studentID == "" {
		respondError(w, http.StatusBadRequest, "student_id is required")
		return
	}

	query := `
		SELECT c.id, c.name, c.description, sc.role, c.faculty_advisor, c.meeting_schedule, c.banner_url
		FROM student_clubs sc
		JOIN clubs c ON sc.club_id = c.id
		WHERE sc.student_id = $1
		ORDER BY c.name ASC;
	`

	rows, err := h.pool.Query(r.Context(), query, studentID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to load enrolled clubs")
		return
	}
	defer rows.Close()

	clubs := make([]models.EnrolledClub, 0)
	for rows.Next() {
		var c models.EnrolledClub
		if err := rows.Scan(&c.ID, &c.Name, &c.Description, &c.Role, &c.FacultyAdvisor, &c.MeetingSchedule, &c.BannerURL); err == nil {
			clubs = append(clubs, c)
		}
	}

	respondJSON(w, http.StatusOK, clubs)
}

// ListCompetitions returns inter-school academic and STEM competitions.
// Route: GET /v1/clubs/competitions?grade={grade}
func (h *ClubsHandler) ListCompetitions(w http.ResponseWriter, r *http.Request) {
	grade := strings.TrimSpace(r.URL.Query().Get("grade"))
	if grade == "" {
		grade = "12"
	}

	query := `
		SELECT id, title, grade, description, deadline, registration_link
		FROM club_competitions
		WHERE grade = $1 OR grade = 'All'
		ORDER BY id ASC;
	`

	rows, err := h.pool.Query(r.Context(), query, grade)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to load competitions")
		return
	}
	defer rows.Close()

	comps := make([]models.Competition, 0)
	for rows.Next() {
		var c models.Competition
		if err := rows.Scan(&c.ID, &c.Title, &c.Grade, &c.Description, &c.Deadline, &c.RegistrationLink); err == nil {
			comps = append(comps, c)
		}
	}

	respondJSON(w, http.StatusOK, comps)
}

// RegisterCompetition records a 1-click event entry registration.
// Route: POST /v1/clubs/competitions/{id}/register
func (h *ClubsHandler) RegisterCompetition(w http.ResponseWriter, r *http.Request) {
	competitionID := chi.URLParam(r, "id")
	if competitionID == "" {
		respondError(w, http.StatusBadRequest, "Competition ID is required")
		return
	}

	var req models.CompetitionRegisterRequest
	_ = json.NewDecoder(r.Body).Decode(&req)

	studentID := strings.TrimSpace(req.StudentID)
	if studentID == "" {
		studentID = resolveStudentID(r)
	}

	if studentID == "" {
		respondError(w, http.StatusBadRequest, "student_id is required")
		return
	}

	regID := fmt.Sprintf("reg-%d", time.Now().UnixNano())
	query := `
		INSERT INTO competition_registrations (id, competition_id, student_id, status, registered_at)
		VALUES ($1, $2, $3, 'REGISTERED', now())
		ON CONFLICT (competition_id, student_id)
		DO UPDATE SET status = 'REGISTERED'
		RETURNING id, competition_id, student_id, status, registered_at;
	`

	var reg models.CompetitionRegistration
	err := h.pool.QueryRow(r.Context(), query, regID, competitionID, studentID).Scan(
		&reg.ID, &reg.CompetitionID, &reg.StudentID, &reg.Status, &reg.RegisteredAt,
	)

	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to register for competition")
		return
	}

	respondJSON(w, http.StatusCreated, map[string]interface{}{
		"message":      "Registration confirmed",
		"registration": reg,
	})
}
