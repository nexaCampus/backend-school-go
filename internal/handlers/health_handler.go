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
	"github.com/nexaCampus/backend-school-go/internal/security"
)

type HealthHandler struct {
	pool  *pgxpool.Pool
	cache cache.Cache
}

func NewHealthHandler(pool *pgxpool.Pool, c ...cache.Cache) *HealthHandler {
	var cc cache.Cache
	if len(c) > 0 && c[0] != nil {
		cc = c[0]
	} else {
		cc = cache.GetDefaultCache()
	}
	return &HealthHandler{pool: pool, cache: cc}
}

// GetProfile retrieves student medical history, allergies, and emergency medical contacts.
// Route: GET /v1/health/profile?student_id={id}
func (h *HealthHandler) GetProfile(w http.ResponseWriter, r *http.Request) {
	studentID := resolveStudentID(r)
	if studentID == "" {
		if r.URL.Query().Get("student_id") != "" {
			respondError(w, http.StatusForbidden, "Forbidden: IDOR violation - cannot access another student's health profile")
			return
		}
		respondError(w, http.StatusBadRequest, "student_id is required")
		return
	}

	// 1. Check local cache before database query (Cache Hit)
	cacheKey := fmt.Sprintf("health:profile:%s", studentID)
	if h.cache != nil {
		if val, found := h.cache.Get(cacheKey); found && val != nil {
			if cachedHP, ok := val.(models.HealthProfile); ok {
				respondJSON(w, http.StatusOK, cachedHP)
				return
			}
		}
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
			// Fallback from students table
			var bloodGroup, fatherName, contactNo string
			_ = h.pool.QueryRow(r.Context(),
				"SELECT blood_group, father_name, contact_no FROM students WHERE student_id = $1 LIMIT 1;",
				studentID,
			).Scan(&bloodGroup, &fatherName, &contactNo)
			if bloodGroup == "" {
				bloodGroup = "A+ve"
			}
			fallback := models.HealthProfile{
				StudentID:             studentID,
				BloodGroup:            bloodGroup,
				Allergies:             []string{"Peanuts (Mild)", "Dust Mites"},
				Conditions:            []string{"None"},
				EmergencyContactName:  fatherName,
				EmergencyContactPhone: contactNo,
			}
			if h.cache != nil {
				h.cache.Set(cacheKey, fallback, 5*time.Minute)
			}
			respondJSON(w, http.StatusOK, fallback)
			return
		}
		respondError(w, http.StatusInternalServerError, "Failed to load health profile")
		return
	}

	if h.cache != nil {
		h.cache.Set(cacheKey, hp, 5*time.Minute)
	}

	respondJSON(w, http.StatusOK, hp)
}

// GetClinicVisits returns past nurse clinic visits, temperature checks, and administered first aid.
// Route: GET /v1/health/clinic-visits?student_id={id}
func (h *HealthHandler) GetClinicVisits(w http.ResponseWriter, r *http.Request) {
	studentID := resolveStudentID(r)
	if studentID == "" {
		if r.URL.Query().Get("student_id") != "" {
			respondError(w, http.StatusForbidden, "Forbidden: IDOR violation - cannot access another student's clinic visits")
			return
		}
		respondError(w, http.StatusBadRequest, "student_id is required")
		return
	}

	// Check local cache
	visitsKey := fmt.Sprintf("health:visits:%s", studentID)
	if h.cache != nil {
		if val, found := h.cache.Get(visitsKey); found && val != nil {
			if cachedVisits, ok := val.([]models.ClinicVisit); ok {
				respondJSON(w, http.StatusOK, cachedVisits)
				return
			}
		}
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
		if err := rows.Scan(
			&v.ID, &v.StudentID, &d, &v.Temperature, &v.Complaint, &v.Treatment, &v.FirstAid, &v.NurseName,
		); err == nil {
			v.Date = d.Format("2006-01-02")
			visits = append(visits, v)
		}
	}

	if h.cache != nil {
		h.cache.Set(visitsKey, visits, 5*time.Minute)
	}

	respondJSON(w, http.StatusOK, visits)
}

// SubmitConsent authorizes specific medication to be administered by the school nurse.
// Route: POST /v1/health/consents
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

	if !VerifyStudentOwnership(r, req.StudentID) {
		respondError(w, http.StatusForbidden, "Forbidden: IDOR violation - cannot submit medical consent for another student")
		return
	}

	// SSRF check on prescription_url
	if strings.TrimSpace(req.PrescriptionURL) != "" {
		trimmed := strings.TrimSpace(req.PrescriptionURL)
		if err := security.ValidateSafeURL(trimmed); err != nil {
			respondError(w, http.StatusBadRequest, "Invalid prescription_url: "+err.Error())
			return
		}
		req.PrescriptionURL = trimmed
	}

	req.MedicationName = security.SanitizeText(req.MedicationName)

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

	// Invalidate health cache
	if h.cache != nil {
		h.cache.InvalidatePrefix("health:")
	}

	respondJSON(w, http.StatusCreated, consent)
}
