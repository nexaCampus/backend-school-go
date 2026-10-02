package models

import (
	"time"
)

// HelpdeskTicket represents an inquiry, doubt, or complaint filed by a student.
type HelpdeskTicket struct {
	ID              string    `json:"id"`
	TicketNo        string    `json:"ticket_no"`
	StudentID       string    `json:"student_id"`
	Department      string    `json:"department"`
	Subject         string    `json:"subject"`
	Message         string    `json:"message"`
	Preview         string    `json:"preview"`
	Status          string    `json:"status"` // SUBMITTED, IN_REVIEW, RESOLVED
	AttachmentURL   *string   `json:"attachment_url,omitempty"`
	ResolutionNotes *string   `json:"resolution_notes,omitempty"`
	Updated         string    `json:"updated"`
	CreatedAt       time.Time `json:"created_at"`
}

// CreateTicketRequest is the payload for POST /v1/helpdesk/tickets
type CreateTicketRequest struct {
	StudentID     string `json:"student_id"`
	Department    string `json:"department"`
	Subject       string `json:"subject"`
	Message       string `json:"message"`
	AttachmentURL string `json:"attachment_url,omitempty"`
}
