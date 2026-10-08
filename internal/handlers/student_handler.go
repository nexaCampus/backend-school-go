package handlers

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nexaCampus/backend-school-go/internal/cache"
	"github.com/nexaCampus/backend-school-go/internal/middleware"
	"github.com/nexaCampus/backend-school-go/internal/models"
	"github.com/nexaCampus/backend-school-go/internal/security"
)

type StudentHandler struct {
	pool      *pgxpool.Pool
	jwtSecret string
	cache     cache.Cache
}

func NewStudentHandler(pool *pgxpool.Pool, jwtSecret string, c ...cache.Cache) *StudentHandler {
	var cc cache.Cache
	if len(c) > 0 && c[0] != nil {
		cc = c[0]
	} else {
		cc = cache.GetDefaultCache()
	}
	return &StudentHandler{pool: pool, jwtSecret: jwtSecret, cache: cc}
}

// GetProfile retrieves a student's full profile, guardian details, and smart card balance.
// Route: GET /v1/students/{id}
func (h *StudentHandler) GetProfile(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		id = resolveStudentID(r)
	}
	if id == "" {
		respondError(w, http.StatusBadRequest, "Student ID is required")
		return
	}

	// Guard against IDOR: Verify token holder is allowed to view this profile
	if claims, ok := middleware.GetClaims(r.Context()); ok && claims != nil {
		if claims.StudentID != id {
			respondError(w, http.StatusForbidden, "Forbidden: IDOR violation - access to another student's profile is denied")
			return
		}
	}

	// 1. Check local cache before database query (Cache Hit: 0 latency)
	cacheKey := fmt.Sprintf("student:profile:%s", id)
	if h.cache != nil {
		if val, found := h.cache.Get(cacheKey); found && val != nil {
			if s, ok := val.(models.Student); ok {
				respondJSON(w, http.StatusOK, s)
				return
			}
		}
	}

	query := `
		SELECT
			id, student_id, dob, full_name, display_name, grade, section, roll_no,
			blood_group, father_name, mother_name, contact_no, email, address,
			school_name, academic_year, avatar_url, smart_card_balance, canteen_balance,
			attendance_pct, streak_days, created_at, updated_at
		FROM students
		WHERE student_id = $1 OR id = $1
		LIMIT 1;
	`

	var student models.Student
	err := h.pool.QueryRow(r.Context(), query, id).Scan(
		&student.ID,
		&student.StudentID,
		&student.DOB,
		&student.FullName,
		&student.DisplayName,
		&student.Grade,
		&student.Section,
		&student.RollNo,
		&student.BloodGroup,
		&student.FatherName,
		&student.MotherName,
		&student.ContactNo,
		&student.Email,
		&student.Address,
		&student.SchoolName,
		&student.AcademicYear,
		&student.AvatarURL,
		&student.SmartCardBalance,
		&student.CanteenBalance,
		&student.AttendancePct,
		&student.StreakDays,
		&student.CreatedAt,
		&student.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			respondError(w, http.StatusNotFound, "Student profile not found")
			return
		}
		respondError(w, http.StatusInternalServerError, "Failed to query student profile")
		return
	}

	// Store in local cache with 5-minute TTL on cache miss
	if h.cache != nil {
		h.cache.Set(cacheKey, student, 5*time.Minute)
	}

	respondJSON(w, http.StatusOK, student)
}

// UpdateAvatar updates a student's profile photo URL.
// Route: PUT /v1/students/{id}/avatar
func (h *StudentHandler) UpdateAvatar(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		id = resolveStudentID(r)
	}
	if id == "" {
		respondError(w, http.StatusBadRequest, "Student ID is required")
		return
	}

	// Guard against IDOR: Verify token holder is allowed to update this avatar
	if claims, ok := middleware.GetClaims(r.Context()); ok && claims != nil {
		if claims.StudentID != id {
			respondError(w, http.StatusForbidden, "Forbidden: IDOR violation - cannot update another student's avatar")
			return
		}
	}

	var req models.UpdateAvatarRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	avatarURL := strings.TrimSpace(req.AvatarURL)
	if avatarURL == "" {
		respondError(w, http.StatusBadRequest, "avatar_url is required")
		return
	}

	// SSRF Prevention: Validate URL scheme and block internal IP addresses
	if err := security.ValidateSafeURL(avatarURL); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid avatar URL: "+err.Error())
		return
	}

	query := `
		UPDATE students
		SET avatar_url = $1, updated_at = now()
		WHERE student_id = $2 OR id = $2
		RETURNING id, student_id, dob, full_name, display_name, grade, section, roll_no,
		          blood_group, father_name, mother_name, contact_no, email, address,
		          school_name, academic_year, avatar_url, smart_card_balance, canteen_balance,
		          attendance_pct, streak_days, created_at, updated_at;
	`

	var student models.Student
	err := h.pool.QueryRow(r.Context(), query, avatarURL, id).Scan(
		&student.ID,
		&student.StudentID,
		&student.DOB,
		&student.FullName,
		&student.DisplayName,
		&student.Grade,
		&student.Section,
		&student.RollNo,
		&student.BloodGroup,
		&student.FatherName,
		&student.MotherName,
		&student.ContactNo,
		&student.Email,
		&student.Address,
		&student.SchoolName,
		&student.AcademicYear,
		&student.AvatarURL,
		&student.SmartCardBalance,
		&student.CanteenBalance,
		&student.AttendancePct,
		&student.StreakDays,
		&student.CreatedAt,
		&student.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			respondError(w, http.StatusNotFound, "Student profile not found")
			return
		}
		respondError(w, http.StatusInternalServerError, "Failed to update profile avatar")
		return
	}

	// Invalidate local cache for updated profile and dashboard
	if h.cache != nil {
		h.cache.Delete(fmt.Sprintf("student:profile:%s", id))
		h.cache.Delete(fmt.Sprintf("student:profile:%s", student.StudentID))
		h.cache.InvalidatePrefix(fmt.Sprintf("student:dashboard:%s", id))
		h.cache.InvalidatePrefix(fmt.Sprintf("student:dashboard:%s", student.StudentID))
	}

	respondJSON(w, http.StatusOK, student)
}

// GetDigitalBadge generates dynamic cryptographic QR badge for gate entry check.
// Route: GET /v1/students/{id}/digital-badge
func (h *StudentHandler) GetDigitalBadge(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		id = resolveStudentID(r)
	}
	if id == "" {
		respondError(w, http.StatusBadRequest, "Student ID is required")
		return
	}

	query := `
		SELECT student_id, full_name, grade, section, roll_no, school_name
		FROM students
		WHERE student_id = $1 OR id = $1
		LIMIT 1;
	`

	var sID, name, grade, section, rollNo, schoolName string
	err := h.pool.QueryRow(r.Context(), query, id).Scan(&sID, &name, &grade, &section, &rollNo, &schoolName)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			respondError(w, http.StatusNotFound, "Student profile not found")
			return
		}
		respondError(w, http.StatusInternalServerError, "Failed to load student for badge")
		return
	}

	// Dynamic valid timestamp: 12 hours from current time
	validUntil := time.Now().Add(12 * time.Hour).Format(time.RFC3339)
	payload := fmt.Sprintf("CAMPUS-GATE|%s|%s|%s|%s|%s", sID, grade, section, rollNo, validUntil)

	// HMAC-SHA256 signature
	mac := hmac.New(sha256.New, []byte(h.jwtSecret))
	mac.Write([]byte(payload))
	signature := hex.EncodeToString(mac.Sum(nil))

	qrPayload := fmt.Sprintf("%s|SIG:%s", payload, signature[:16])

	respondJSON(w, http.StatusOK, models.DigitalBadgeResponse{
		StudentID:  sID,
		Name:       name,
		Grade:      grade,
		Section:    section,
		RollNo:     rollNo,
		SchoolName: schoolName,
		QRCodeData: qrPayload,
		ValidUntil: validUntil,
		Signature:  signature,
	})
}

// DashboardSummaryResponse is the unified response for the home dashboard.
type DashboardSummaryResponse struct {
	Student              models.Student         `json:"student"`
	AttendanceRate       float64                `json:"attendance_rate"`
	TodayClasses         []models.TimetableSlot `json:"today_classes"`
	PendingHomeworkCount int                    `json:"pending_homework_count"`
	UrgentNotices        []models.Notice        `json:"urgent_notices"`
}

// GetDashboardSummary aggregates today's classes, attendance rate, pending homework count, and notices.
// Route: GET /v1/dashboard/summary?student_id={id}&grade={grade}&section={section}
func (h *StudentHandler) GetDashboardSummary(w http.ResponseWriter, r *http.Request) {
	studentID := resolveStudentID(r)
	if studentID == "" {
		respondError(w, http.StatusBadRequest, "student_id is required")
		return
	}

	grade := strings.TrimSpace(r.URL.Query().Get("grade"))
	section := strings.TrimSpace(r.URL.Query().Get("section"))

	// Check local cache before executing 4 database queries (Cache Hit)
	cacheKey := fmt.Sprintf("student:dashboard:%s:%s:%s", studentID, grade, section)
	if h.cache != nil {
		if val, found := h.cache.Get(cacheKey); found && val != nil {
			if cachedSummary, ok := val.(DashboardSummaryResponse); ok {
				respondJSON(w, http.StatusOK, cachedSummary)
				return
			}
		}
	}

	// 1. Fetch Student Profile
	var student models.Student
	err := h.pool.QueryRow(r.Context(), `
		SELECT
			id, student_id, dob, full_name, display_name, grade, section, roll_no,
			blood_group, father_name, mother_name, contact_no, email, address,
			school_name, academic_year, avatar_url, smart_card_balance, canteen_balance,
			attendance_pct, streak_days, created_at, updated_at
		FROM students
		WHERE student_id = $1 OR id = $1
		LIMIT 1;
	`, studentID).Scan(
		&student.ID,
		&student.StudentID,
		&student.DOB,
		&student.FullName,
		&student.DisplayName,
		&student.Grade,
		&student.Section,
		&student.RollNo,
		&student.BloodGroup,
		&student.FatherName,
		&student.MotherName,
		&student.ContactNo,
		&student.Email,
		&student.Address,
		&student.SchoolName,
		&student.AcademicYear,
		&student.AvatarURL,
		&student.SmartCardBalance,
		&student.CanteenBalance,
		&student.AttendancePct,
		&student.StreakDays,
		&student.CreatedAt,
		&student.UpdatedAt,
	)

	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		respondError(w, http.StatusInternalServerError, "Failed to load student for dashboard")
		return
	}

	if grade == "" && student.Grade != "" {
		grade = student.Grade
	}
	if section == "" && student.Section != "" {
		section = student.Section
	}

	// 2. Fetch Today's Classes
	dayOfWeek := int(time.Now().Weekday())
	if dayOfWeek == 0 {
		dayOfWeek = 1
	}

	rows, err := h.pool.Query(r.Context(), `
		SELECT id, grade, section, day_of_week, period_label, time_label, subject, detail, room_number, teacher_name, is_live
		FROM timetable
		WHERE grade = $1 AND section = $2 AND day_of_week = $3
		ORDER BY id ASC;
	`, grade, section, dayOfWeek)

	todayClasses := make([]models.TimetableSlot, 0)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var slot models.TimetableSlot
			if err := rows.Scan(
				&slot.ID, &slot.Grade, &slot.Section, &slot.DayOfWeek,
				&slot.PeriodLabel, &slot.Time, &slot.Subject, &slot.Detail,
				&slot.RoomNumber, &slot.Teacher, &slot.IsLive,
			); err == nil {
				todayClasses = append(todayClasses, slot)
			}
		}
	}

	// 3. Pending Homework Count
	var pendingHomeworkCount int
	_ = h.pool.QueryRow(r.Context(), `
		SELECT count(*)
		FROM homework h
		LEFT JOIN homework_submissions hs
			ON h.id = hs.homework_id AND hs.student_id = $1
		WHERE h.grade = $2 AND h.section = $3
		  AND (hs.status IS NULL OR hs.status = 'pending');
	`, studentID, grade, section).Scan(&pendingHomeworkCount)

	// 4. Urgent Notices
	noticeRows, err := h.pool.Query(r.Context(), `
		SELECT id, category, title, body, date_label, author, is_urgent, attachment_name, attachment_url, published_at
		FROM notices
		ORDER BY is_urgent DESC, published_at DESC
		LIMIT 5;
	`)

	urgentNotices := make([]models.Notice, 0)
	if err == nil {
		defer noticeRows.Close()
		for noticeRows.Next() {
			var n models.Notice
			if err := noticeRows.Scan(
				&n.ID, &n.Category, &n.Title, &n.Body, &n.Date,
				&n.Author, &n.IsUrgent, &n.AttachmentName, &n.AttachmentURL, &n.PublishedAt,
			); err == nil {
				urgentNotices = append(urgentNotices, n)
			}
		}
	}

	attendanceRate := float64(student.AttendancePct)
	if attendanceRate <= 0 {
		attendanceRate = 96.0
	}

	summary := DashboardSummaryResponse{
		Student:              student,
		AttendanceRate:       attendanceRate,
		TodayClasses:         todayClasses,
		PendingHomeworkCount: pendingHomeworkCount,
		UrgentNotices:        urgentNotices,
	}

	if h.cache != nil {
		h.cache.Set(cacheKey, summary, 2*time.Minute)
	}

	respondJSON(w, http.StatusOK, summary)
}
