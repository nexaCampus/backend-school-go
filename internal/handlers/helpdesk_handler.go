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

type HelpdeskHandler struct {
	pool *pgxpool.Pool
}

func NewHelpdeskHandler(pool *pgxpool.Pool) *HelpdeskHandler {
	return &HelpdeskHandler{pool: pool}
}

// CreateTicket logs a new inquiry or complaint ticket.
// Route: POST /v1/helpdesk/tickets
func (h *HelpdeskHandler) CreateTicket(w http.ResponseWriter, r *http.Request) {
	var req models.CreateTicketRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	req.StudentID = strings.TrimSpace(req.StudentID)
	if req.StudentID == "" {
		req.StudentID = resolveStudentID(r)
	}

	if req.StudentID == "" || req.Department == "" || req.Subject == "" || req.Message == "" {
		respondError(w, http.StatusBadRequest, "student_id, department, subject, and message are required")
		return
	}

	ticketID := fmt.Sprintf("tkt-%d", time.Now().UnixNano())
	ticketNo := fmt.Sprintf("TKT-%04d", (time.Now().UnixNano()/1000)%9000+1000)

	preview := req.Message
	if len(preview) > 60 {
		preview = preview[:57] + "..."
	}

	var attach *string
	if strings.TrimSpace(req.AttachmentURL) != "" {
		trimmed := strings.TrimSpace(req.AttachmentURL)
		attach = &trimmed
	}

	query := `
		INSERT INTO helpdesk_tickets (
			id, ticket_no, student_id, department, subject, message,
			preview, status, attachment_url, updated_label, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, 'SUBMITTED', $8, 'Expected reply within 24 hours', now())
		RETURNING id, ticket_no, student_id, department, subject, message, preview, status, attachment_url, resolution_notes, updated_label, created_at;
	`

	var tkt models.HelpdeskTicket
	err := h.pool.QueryRow(r.Context(), query,
		ticketID, ticketNo, req.StudentID, req.Department, req.Subject, req.Message, preview, attach,
	).Scan(
		&tkt.ID, &tkt.TicketNo, &tkt.StudentID, &tkt.Department, &tkt.Subject, &tkt.Message,
		&tkt.Preview, &tkt.Status, &tkt.AttachmentURL, &tkt.ResolutionNotes, &tkt.Updated, &tkt.CreatedAt,
	)

	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to submit ticket")
		return
	}

	respondJSON(w, http.StatusCreated, tkt)
}

// ListTickets retrieves submitted helpdesk tickets for a student.
// Route: GET /v1/helpdesk/tickets?student_id={id}
func (h *HelpdeskHandler) ListTickets(w http.ResponseWriter, r *http.Request) {
	studentID := resolveStudentID(r)
	if studentID == "" {
		respondError(w, http.StatusBadRequest, "student_id is required")
		return
	}

	query := `
		SELECT
			id, ticket_no, student_id, department, subject, message,
			preview, status, attachment_url, resolution_notes, updated_label, created_at
		FROM helpdesk_tickets
		WHERE student_id = $1
		ORDER BY created_at DESC;
	`

	rows, err := h.pool.Query(r.Context(), query, studentID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to fetch helpdesk tickets")
		return
	}
	defer rows.Close()

	tickets := make([]models.HelpdeskTicket, 0)
	for rows.Next() {
		var tkt models.HelpdeskTicket
		if err := rows.Scan(
			&tkt.ID, &tkt.TicketNo, &tkt.StudentID, &tkt.Department, &tkt.Subject, &tkt.Message,
			&tkt.Preview, &tkt.Status, &tkt.AttachmentURL, &tkt.ResolutionNotes, &tkt.Updated, &tkt.CreatedAt,
		); err == nil {
			tickets = append(tickets, tkt)
		}
	}

	respondJSON(w, http.StatusOK, tickets)
}
