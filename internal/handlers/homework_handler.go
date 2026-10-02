package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nexaCampus/backend-school-go/internal/models"
)

type HomeworkHandler struct {
	pool *pgxpool.Pool
}

func NewHomeworkHandler(pool *pgxpool.Pool) *HomeworkHandler {
	return &HomeworkHandler{pool: pool}
}

// ListHomework filters homework assignments with student submission status.
// Route: GET /v1/homework?grade={grade}&section={section}&status={all|pending|submitted}
func (h *HomeworkHandler) ListHomework(w http.ResponseWriter, r *http.Request) {
	studentID := resolveStudentID(r)
	grade := strings.TrimSpace(r.URL.Query().Get("grade"))
	section := strings.TrimSpace(r.URL.Query().Get("section"))
	statusFilter := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("status")))

	if (grade == "" || section == "") && studentID != "" {
		_ = h.pool.QueryRow(r.Context(),
			"SELECT grade, section FROM students WHERE student_id = $1 LIMIT 1;",
			studentID,
		).Scan(&grade, &section)
	}

	if grade == "" {
		grade = "12"
	}
	if section == "" {
		section = "E"
	}

	query := `
		SELECT
			h.id, h.grade, h.section, h.subject, h.unit, h.title, h.teacher,
			h.due, h.due_date, h.description, h.attachment_name, h.attachment_meta,
			h.max_score, h.created_at,
			hs.status, hs.submission_file_url, hs.submitted_at
		FROM homework h
		LEFT JOIN homework_submissions hs
			ON h.id = hs.homework_id AND hs.student_id = $1
		WHERE h.grade = $2 AND h.section = $3
		ORDER BY h.due_date ASC NULLS LAST, h.created_at DESC;
	`

	rows, err := h.pool.Query(r.Context(), query, studentID, grade, section)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to query homework assignments")
		return
	}
	defer rows.Close()

	items := make([]models.Homework, 0)
	for rows.Next() {
		var hw models.Homework
		var subStatus *string
		var subFile *string
		var submittedAt *time.Time

		err := rows.Scan(
			&hw.ID, &hw.Grade, &hw.Section, &hw.Subject, &hw.Unit, &hw.Title, &hw.Teacher,
			&hw.Due, &hw.DueDate, &hw.Description, &hw.AttachmentName, &hw.AttachmentMeta,
			&hw.MaxScore, &hw.CreatedAt,
			&subStatus, &subFile, &submittedAt,
		)
		if err != nil {
			continue
		}

		if subStatus != nil && *subStatus != "" {
			hw.Status = strings.ToUpper(*subStatus)
			hw.SubmissionFileURL = subFile
			hw.SubmittedAt = submittedAt
			hw.ProgressPct = 100
		} else {
			hw.Status = "PENDING"
			hw.ProgressPct = 0
		}

		if statusFilter != "" && statusFilter != "all" {
			if strings.ToLower(hw.Status) != statusFilter {
				continue
			}
		}

		items = append(items, hw)
	}

	respondJSON(w, http.StatusOK, items)
}

// GetHomeworkDetail retrieves full assignment instructions, reference files, and teacher attachments.
// Route: GET /v1/homework/{id}
func (h *HomeworkHandler) GetHomeworkDetail(w http.ResponseWriter, r *http.Request) {
	homeworkID := chi.URLParam(r, "id")
	if homeworkID == "" {
		respondError(w, http.StatusBadRequest, "Homework ID is required")
		return
	}

	studentID := resolveStudentID(r)

	query := `
		SELECT
			h.id, h.grade, h.section, h.subject, h.unit, h.title, h.teacher,
			h.due, h.due_date, h.description, h.attachment_name, h.attachment_meta,
			h.max_score, h.created_at,
			hs.status, hs.submission_file_url, hs.submitted_at
		FROM homework h
		LEFT JOIN homework_submissions hs
			ON h.id = hs.homework_id AND hs.student_id = $1
		WHERE h.id = $2
		LIMIT 1;
	`

	var hw models.Homework
	var subStatus *string
	var subFile *string
	var submittedAt *time.Time

	err := h.pool.QueryRow(r.Context(), query, studentID, homeworkID).Scan(
		&hw.ID, &hw.Grade, &hw.Section, &hw.Subject, &hw.Unit, &hw.Title, &hw.Teacher,
		&hw.Due, &hw.DueDate, &hw.Description, &hw.AttachmentName, &hw.AttachmentMeta,
		&hw.MaxScore, &hw.CreatedAt,
		&subStatus, &subFile, &submittedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			respondError(w, http.StatusNotFound, "Homework assignment not found")
			return
		}
		respondError(w, http.StatusInternalServerError, "Failed to load homework details")
		return
	}

	if subStatus != nil && *subStatus != "" {
		hw.Status = strings.ToUpper(*subStatus)
		hw.SubmissionFileURL = subFile
		hw.SubmittedAt = submittedAt
		hw.ProgressPct = 100
	} else {
		hw.Status = "PENDING"
		hw.ProgressPct = 0
	}

	respondJSON(w, http.StatusOK, hw)
}

// SubmitHomework records a student homework submission.
// Route: POST /v1/homework/{id}/submit
func (h *HomeworkHandler) SubmitHomework(w http.ResponseWriter, r *http.Request) {
	homeworkID := chi.URLParam(r, "id")
	if homeworkID == "" {
		respondError(w, http.StatusBadRequest, "Homework ID is required in URL path")
		return
	}

	var req models.SubmitHomeworkRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	req.StudentID = strings.TrimSpace(req.StudentID)
	if req.StudentID == "" {
		req.StudentID = resolveStudentID(r)
	}

	if req.StudentID == "" || req.SubmissionFileURL == "" {
		respondError(w, http.StatusBadRequest, "student_id and submission_file_url are required")
		return
	}

	submissionID := fmt.Sprintf("sub-%d", time.Now().UnixNano())
	query := `
		INSERT INTO homework_submissions (id, homework_id, student_id, submission_file_url, status, submitted_at, updated_at)
		VALUES ($1, $2, $3, $4, 'submitted', now(), now())
		ON CONFLICT (homework_id, student_id)
		DO UPDATE SET
			submission_file_url = EXCLUDED.submission_file_url,
			status = 'submitted',
			updated_at = now()
		RETURNING id, homework_id, student_id, submission_file_url, status, submitted_at, updated_at;
	`

	var sub models.HomeworkSubmission
	err := h.pool.QueryRow(r.Context(), query, submissionID, homeworkID, req.StudentID, req.SubmissionFileURL).Scan(
		&sub.ID, &sub.HomeworkID, &sub.StudentID, &sub.SubmissionFileURL, &sub.Status, &sub.SubmittedAt, &sub.UpdatedAt,
	)

	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to record homework submission")
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"message":    "Homework submitted successfully",
		"submission": sub,
	})
}
