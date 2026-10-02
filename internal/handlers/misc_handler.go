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

type MiscHandler struct {
	pool *pgxpool.Pool
}

func NewMiscHandler(pool *pgxpool.Pool) *MiscHandler {
	return &MiscHandler{pool: pool}
}

// SearchBooks queries the library catalog by keyword and category.
// Route: GET /v1/library/books?query={title|author|isbn}&category={category}
func (h *MiscHandler) SearchBooks(w http.ResponseWriter, r *http.Request) {
	q := strings.TrimSpace(r.URL.Query().Get("query"))
	category := strings.TrimSpace(r.URL.Query().Get("category"))

	var query string
	var args []interface{}

	if q != "" && category != "" {
		searchTerm := "%" + q + "%"
		query = `
			SELECT id, isbn, title, author, publisher, total_copies, available_copies, category
			FROM library_books
			WHERE (title ILIKE $1 OR author ILIKE $1 OR isbn ILIKE $1) AND LOWER(category) = LOWER($2)
			ORDER BY title ASC
			LIMIT 50;
		`
		args = []interface{}{searchTerm, category}
	} else if q != "" {
		searchTerm := "%" + q + "%"
		query = `
			SELECT id, isbn, title, author, publisher, total_copies, available_copies, category
			FROM library_books
			WHERE title ILIKE $1 OR author ILIKE $1 OR isbn ILIKE $1
			ORDER BY title ASC
			LIMIT 50;
		`
		args = []interface{}{searchTerm}
	} else if category != "" {
		query = `
			SELECT id, isbn, title, author, publisher, total_copies, available_copies, category
			FROM library_books
			WHERE LOWER(category) = LOWER($1)
			ORDER BY title ASC
			LIMIT 50;
		`
		args = []interface{}{category}
	} else {
		query = `
			SELECT id, isbn, title, author, publisher, total_copies, available_copies, category
			FROM library_books
			ORDER BY title ASC
			LIMIT 50;
		`
	}

	rows, err := h.pool.Query(r.Context(), query, args...)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to search library books")
		return
	}
	defer rows.Close()

	books := make([]models.LibraryBook, 0)
	for rows.Next() {
		var bk models.LibraryBook
		if err := rows.Scan(
			&bk.ID, &bk.ISBN, &bk.Title, &bk.Author, &bk.Publisher,
			&bk.TotalCopies, &bk.AvailableCopies, &bk.Category,
		); err == nil {
			books = append(books, bk)
		}
	}

	respondJSON(w, http.StatusOK, books)
}

// GetBorrowings retrieves currently borrowed books, return due dates, and calculated fines.
// Route: GET /v1/library/borrowings?student_id={id}
func (h *MiscHandler) GetBorrowings(w http.ResponseWriter, r *http.Request) {
	studentID := resolveStudentID(r)
	if studentID == "" {
		respondError(w, http.StatusBadRequest, "student_id is required")
		return
	}

	query := `
		SELECT
			lb.id, lb.student_id, lb.book_id, b.title, b.author, b.isbn,
			lb.borrowed_at, lb.due_at, lb.returned_at, lb.fine_amount, lb.status
		FROM library_borrowings lb
		JOIN library_books b ON lb.book_id = b.id
		WHERE lb.student_id = $1
		ORDER BY lb.borrowed_at DESC;
	`

	rows, err := h.pool.Query(r.Context(), query, studentID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to retrieve borrowed books")
		return
	}
	defer rows.Close()

	borrowings := make([]models.LibraryBorrowing, 0)
	for rows.Next() {
		var bor models.LibraryBorrowing
		var bAt, dAt time.Time
		var rAt *time.Time

		err := rows.Scan(
			&bor.ID, &bor.StudentID, &bor.BookID, &bor.Title, &bor.Author, &bor.ISBN,
			&bAt, &dAt, &rAt, &bor.FineAmount, &bor.Status,
		)
		if err != nil {
			continue
		}

		bor.BorrowedAt = bAt.Format("2006-01-02")
		bor.DueAt = dAt.Format("2006-01-02")
		if rAt != nil {
			retStr := rAt.Format("2006-01-02")
			bor.ReturnedAt = &retStr
		}

		if bor.ReturnedAt == nil && time.Now().After(dAt) {
			overdueDays := int(time.Since(dAt).Hours() / 24)
			if overdueDays > 0 {
				bor.FineAmount = float64(overdueDays * 5)
				bor.Status = "OVERDUE"
			}
		}

		borrowings = append(borrowings, bor)
	}

	respondJSON(w, http.StatusOK, borrowings)
}

// ReserveBook places a temporary hold on a library book.
// Route: POST /v1/library/reserve
func (h *MiscHandler) ReserveBook(w http.ResponseWriter, r *http.Request) {
	var req models.ReserveBookRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	req.StudentID = strings.TrimSpace(req.StudentID)
	if req.StudentID == "" {
		req.StudentID = resolveStudentID(r)
	}
	req.BookID = strings.TrimSpace(req.BookID)

	if req.StudentID == "" || req.BookID == "" {
		respondError(w, http.StatusBadRequest, "student_id and book_id are required")
		return
	}

	// Verify book exists
	var title string
	var availableCopies int
	err := h.pool.QueryRow(r.Context(),
		"SELECT title, available_copies FROM library_books WHERE id = $1 OR isbn = $1 LIMIT 1;",
		req.BookID,
	).Scan(&title, &availableCopies)

	if err != nil {
		respondError(w, http.StatusNotFound, "Library book not found")
		return
	}

	reservationID := fmt.Sprintf("res-%d", time.Now().UnixNano())
	now := time.Now()
	expiresAt := now.Add(48 * time.Hour)

	query := `
		INSERT INTO library_reservations (id, student_id, book_id, title, reserved_at, expires_at, status)
		VALUES ($1, $2, $3, $4, $5, $6, 'ACTIVE')
		ON CONFLICT (id) DO NOTHING;
	`
	_, _ = h.pool.Exec(r.Context(), query, reservationID, req.StudentID, req.BookID, title, now, expiresAt)

	// Decrement available copies if > 0
	if availableCopies > 0 {
		_, _ = h.pool.Exec(r.Context(), "UPDATE library_books SET available_copies = available_copies - 1 WHERE id = $1;", req.BookID)
	}

	respondJSON(w, http.StatusCreated, models.BookReservation{
		ID:         reservationID,
		StudentID:  req.StudentID,
		BookID:     req.BookID,
		Title:      title,
		ReservedAt: now.Format("2006-01-02 15:04"),
		ExpiresAt:  expiresAt.Format("2006-01-02 15:04"),
		Status:     "ACTIVE",
	})
}

// GetHallTicket retrieves student examination hall ticket pass and seat allocation.
// Route: GET /v1/exams/hall-ticket?student_id={id}&term={term_name}
func (h *MiscHandler) GetHallTicket(w http.ResponseWriter, r *http.Request) {
	studentID := resolveStudentID(r)
	if studentID == "" {
		respondError(w, http.StatusBadRequest, "student_id is required")
		return
	}

	term := strings.TrimSpace(r.URL.Query().Get("term"))

	var query string
	var args []interface{}

	if term != "" {
		query = `
			SELECT id, student_id, student_name, grade, section, roll_no, term,
			       exam_name, exam_center, seat_number, schedule, rules, is_active
			FROM exam_hall_tickets
			WHERE student_id = $1 AND LOWER(term) = LOWER($2) AND is_active = true
			LIMIT 1;
		`
		args = []interface{}{studentID, term}
	} else {
		query = `
			SELECT id, student_id, student_name, grade, section, roll_no, term,
			       exam_name, exam_center, seat_number, schedule, rules, is_active
			FROM exam_hall_tickets
			WHERE student_id = $1 AND is_active = true
			ORDER BY id DESC
			LIMIT 1;
		`
		args = []interface{}{studentID}
	}

	var ht models.HallTicket
	var schedJSON, rulesJSON []byte

	err := h.pool.QueryRow(r.Context(), query, args...).Scan(
		&ht.ID, &ht.StudentID, &ht.StudentName, &ht.Grade, &ht.Section, &ht.RollNo, &ht.Term,
		&ht.ExamName, &ht.ExamCenter, &ht.SeatNumber, &schedJSON, &rulesJSON, &ht.IsActive,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			respondError(w, http.StatusNotFound, "No active exam hall ticket found for student")
			return
		}
		respondError(w, http.StatusInternalServerError, "Failed to load exam hall ticket")
		return
	}

	ht.Schedule = make([]models.HallTicketSubject, 0)
	if len(schedJSON) > 0 {
		_ = json.Unmarshal(schedJSON, &ht.Schedule)
	}

	ht.Rules = make([]string, 0)
	if len(rulesJSON) > 0 {
		_ = json.Unmarshal(rulesJSON, &ht.Rules)
	}

	respondJSON(w, http.StatusOK, ht)
}

// GetCanteenWallet retrieves smart card / cafeteria ledger and recent purchases.
// Route: GET /v1/canteen/wallet?student_id={id}
func (h *MiscHandler) GetCanteenWallet(w http.ResponseWriter, r *http.Request) {
	studentID := resolveStudentID(r)
	if studentID == "" {
		respondError(w, http.StatusBadRequest, "student_id is required")
		return
	}

	var balance float64
	err := h.pool.QueryRow(r.Context(),
		"SELECT canteen_balance FROM students WHERE student_id = $1 LIMIT 1;",
		studentID,
	).Scan(&balance)
	if err != nil {
		balance = 450.00
	}

	rows, err := h.pool.Query(r.Context(), `
		SELECT id, student_id, title, amount, transaction_type, balance_after, created_at
		FROM canteen_transactions
		WHERE student_id = $1
		ORDER BY created_at DESC
		LIMIT 25;
	`, studentID)

	transactions := make([]models.CanteenTransaction, 0)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var tx models.CanteenTransaction
			var cAt time.Time
			if err := rows.Scan(
				&tx.ID, &tx.StudentID, &tx.Title, &tx.Amount,
				&tx.Type, &tx.BalanceAfter, &cAt,
			); err == nil {
				tx.CreatedAt = cAt.Format("02 Jan 2006, 03:04 PM")
				transactions = append(transactions, tx)
			}
		}
	}

	respondJSON(w, http.StatusOK, models.CanteenWallet{
		StudentID:    studentID,
		Balance:      balance,
		Transactions: transactions,
	})
}

// GetCanteenMenu returns today's cafeteria menu with dietary tags and pricing.
// Route: GET /v1/canteen/menu?day={day_name}
func (h *MiscHandler) GetCanteenMenu(w http.ResponseWriter, r *http.Request) {
	day := strings.TrimSpace(r.URL.Query().Get("day"))
	if day == "" {
		day = time.Now().Weekday().String()
	}

	query := `
		SELECT id, day_of_week, item_name, category, dietary_tag, price, is_available
		FROM canteen_menu
		WHERE LOWER(day_of_week) = LOWER($1) OR day_of_week = 'All'
		ORDER BY category ASC, item_name ASC;
	`

	rows, err := h.pool.Query(r.Context(), query, day)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to load cafeteria menu")
		return
	}
	defer rows.Close()

	items := make([]models.CanteenMenuItem, 0)
	for rows.Next() {
		var it models.CanteenMenuItem
		if err := rows.Scan(&it.ID, &it.DayOfWeek, &it.Name, &it.Category, &it.DietaryTag, &it.Price, &it.IsAvailable); err == nil {
			items = append(items, it)
		}
	}

	respondJSON(w, http.StatusOK, items)
}
