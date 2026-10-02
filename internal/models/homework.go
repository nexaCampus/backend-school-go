package models

import (
	"time"
)

// Homework represents an academic assignment assigned to students.
type Homework struct {
	ID                string     `json:"id"`
	Grade             string     `json:"grade"`
	Section           string     `json:"section"`
	Subject           string     `json:"subject"`
	Unit              string     `json:"unit"`
	Title             string     `json:"title"`
	Teacher           string     `json:"teacher"`
	Due               string     `json:"due"`
	DueDate           *time.Time `json:"due_date,omitempty"`
	Description       string     `json:"description"`
	Status            string     `json:"status"` // PENDING, SUBMITTED, NOT_STARTED, EVALUATED
	ProgressPct       int        `json:"progress_pct"`
	AttachmentName    *string    `json:"attachment_name,omitempty"`
	AttachmentMeta    *string    `json:"attachment_meta,omitempty"`
	MaxScore          float64    `json:"max_score"`
	SubmissionFileURL *string    `json:"submission_file_url,omitempty"`
	SubmittedAt       *time.Time `json:"submitted_at,omitempty"`
	CreatedAt         time.Time  `json:"created_at"`
}

// HomeworkSubmission represents student solution submissions.
type HomeworkSubmission struct {
	ID                string     `json:"id"`
	HomeworkID        string     `json:"homework_id"`
	StudentID         string     `json:"student_id"`
	SubmissionFileURL string     `json:"submission_file_url"`
	Status            string     `json:"status"` // submitted, evaluated
	ObtainedScore     *float64   `json:"obtained_score,omitempty"`
	Feedback          *string    `json:"feedback,omitempty"`
	SubmittedAt       time.Time  `json:"submitted_at"`
	UpdatedAt         *time.Time `json:"updated_at,omitempty"`
}

// SubmitHomeworkRequest is the payload for POST /v1/homework/{id}/submit
type SubmitHomeworkRequest struct {
	StudentID         string `json:"student_id"`
	SubmissionFileURL string `json:"submission_file_url"`
}
