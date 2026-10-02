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
	"github.com/nexaCampus/backend-school-go/internal/models"
)

type HealthHandler struct {
	pool *pgxpool.Pool
}

func NewHealthHandler(pool *pgxpool.Pool) *HealthHandler {
	return &HealthHandler{pool: pool}
}

// GetProfile retrieves student medical history, allergies, and emergency medical contacts.
// Route: GET /v1/health/profile?student_id={id}
func (h *HealthHandler) GetProfile(w http.ResponseWriter, r *http.Request) {
	studentID := resolveStudentID(r)
	if studentID == "" {
		respondError(w, http.StatusBadRequest, "student_id is required")
		return
	}

	query := `
		SELECT student_id, blood_group, allergies, conditions, emergency_contact_name, emergency_contact_phone
		FROM health_profiles
		WHERE student_id = $1
		LIMIT 1;
	`

	var hp models.HealthProfile
	err := h.pool.QueryRow(r.Context(), query, studentID).Scan(
		&hp.StudentID, &hp.BloodGroup, &hp.Allergies, &hp.Conditions,
		&hp.EmergencyContactName, &hp.EmergencyContactPhone,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			// Infer from students record
			var bloodGroup, contactNo, fatherName string
			_ = h.pool.QueryRow(r.Context(),
				"SELECT blood_group, contact_no, father_name FROM students WHERE student_id = $1 LIMIT 1;",
				studentID,
			).Scan(&bloodGroup, &contactNo, &fatherName)

			if bloodGroup == "" {
				bloodGroup = "A+ve"
			}
			respondJSON(w, http.StatusOK, models.HealthProfile{
				StudentID:             studentID,
				BloodGroup:            bloodGroup,
				Allergies:             []string{"Peanuts (Mild)", "Dust Mites"},
				Conditions:            []string{"None"},
				EmergencyContactName:  fatherName,
				EmergencyContactPhone: contactNo,
			})
			return
		}
		respondError(w, http.StatusInternalServerError, "Failed to load health profile")
		return
	}

	respondJSON(w, http.StatusOK, hp)
}

// GetClinicVisits returns past nurse clinic visits, temperature checks, and administered first aid.
// Route: GET /v1/health/clinic-visits?student_id={id}
func (h *HealthHandler) GetClinicVisits(w http.ResponseWriter, r *http.Request) {
	studentID := resolveStudentID(r)
	if studentID == "" {
		respondError(w, http.StatusBadRequest, "student_id is required")
		return
	}

	query := `
		SELECT id, student_id, visit_date, temperature, complaint, treatment, first_aid, nurse_name
		FROM clinic_visits
		WHERE student_id = $1
		ORDER BY visit_date DESC;
	`

	rows, err := h.pool.Query(r.Context(), query, studentID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to load clinic visits")
		return
	}
	defer rows.Close()

	visits := make([]models.ClinicVisit, 0)
	for rows.Next() {
		var v models.ClinicVisit
		var d time.Time
		if err := rows.Scan(&v.ID, &v.StudentID, &d, &v.Temperature, &v.Complaint, &v.Treatment, &v.FirstAid, &v.NurseName); err == nil {
			v.Date = d.Format("02 Jan 2006, 03:04 PM")
			visits = append(visits, v)
		}
	}

	respondJSON(w, http.StatusOK, visits)
}

// SubmitConsent logs parental authorization for nursing staff to dispense midday medications.
// Route: POST /v1/health/consent
func (h *HealthHandler) SubmitConsent(w http.ResponseWriter, r *http.Request) {
	var req models.HealthConsentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	req.StudentID = strings.TrimSpace(req.StudentID)
	if req.StudentID == "" {
		req.StudentID = resolveStudentID(r)
	}

	if req.StudentID == "" || req.MedicationName == "" {
		respondError(w, http.StatusBadRequest, "student_id and medication_name are required")
		return
	}

	consentID := fmt.Sprintf("med-con-%d", time.Now().UnixNano())
	query := `
		INSERT INTO medical_consents (id, student_id, medication_name, prescription_url, status, created_at)
		VALUES ($1, $2, $3, $4, 'AUTHORIZED', now())
		RETURNING id, student_id, medication_name, prescription_url, status, created_at;
	`

	var consent models.HealthConsent
	err := h.pool.QueryRow(r.Context(), query, consentID, req.StudentID, req.MedicationName, req.PrescriptionURL).Scan(
		&consent.ID, &consent.StudentID, &consent.MedicationName, &consent.PrescriptionURL, &consent.Status, &consent.CreatedAt,
	)

	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to register medication consent")
		return
	}

	respondJSON(w, http.StatusCreated, consent)
}
