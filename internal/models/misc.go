package models

// LibraryBook represents a catalogued volume in the school library.
type LibraryBook struct {
	ID              string `json:"id"`
	ISBN            string `json:"isbn"`
	Title           string `json:"title"`
	Author          string `json:"author"`
	Publisher       string `json:"publisher,omitempty"`
	TotalCopies     int    `json:"total_copies"`
	AvailableCopies int    `json:"available_copies"`
	Category        string `json:"category,omitempty"`
}

// LibraryBorrowing represents an active or past book lending record.
type LibraryBorrowing struct {
	ID         string  `json:"id"`
	StudentID  string  `json:"student_id"`
	BookID     string  `json:"book_id"`
	Title      string  `json:"title"`
	Author     string  `json:"author"`
	ISBN       string  `json:"isbn"`
	BorrowedAt string  `json:"borrowed_at"`
	DueAt      string  `json:"due_at"`
	ReturnedAt *string `json:"returned_at,omitempty"`
	FineAmount float64 `json:"fine_amount"`
	Status     string  `json:"status"` // BORROWED, OVERDUE, RETURNED
}

// ReserveBookRequest is the payload for POST /v1/library/reserve
type ReserveBookRequest struct {
	StudentID string `json:"student_id"`
	BookID    string `json:"book_id"`
}

// BookReservation represents an active hold placed on a library book.
type BookReservation struct {
	ID         string `json:"id"`
	StudentID  string `json:"student_id"`
	BookID     string `json:"book_id"`
	Title      string `json:"title"`
	ReservedAt string `json:"reserved_at"`
	ExpiresAt  string `json:"expires_at"`
	Status     string `json:"status"`
}

// HallTicketSubject represents a subject entry in an examination schedule.
type HallTicketSubject struct {
	Subject string `json:"subject"`
	Date    string `json:"date"`
	Time    string `json:"time"`
	Room    string `json:"room"`
}

// HallTicket represents an authorized examination entry pass and seat allocation.
type HallTicket struct {
	ID          string              `json:"id"`
	StudentID   string              `json:"student_id"`
	StudentName string              `json:"student_name"`
	Grade       string              `json:"grade"`
	Section     string              `json:"section"`
	RollNo      string              `json:"roll_no"`
	Term        string              `json:"term"`
	ExamName    string              `json:"exam_name"`
	ExamCenter  string              `json:"exam_center"`
	SeatNumber  string              `json:"seat_number"`
	Schedule    []HallTicketSubject `json:"schedule"`
	Rules       []string            `json:"rules"`
	IsActive    bool                `json:"is_active"`
}

// CanteenTransaction represents a debit or credit against the cafeteria wallet.
type CanteenTransaction struct {
	ID           string  `json:"id"`
	StudentID    string  `json:"student_id,omitempty"`
	Title        string  `json:"title"`
	Amount       float64 `json:"amount"`
	Type         string  `json:"type"` // DEBIT, CREDIT
	BalanceAfter float64 `json:"balance_after"`
	CreatedAt    string  `json:"created_at"`
}

// CanteenWallet represents smart card / cafeteria ledger and recent purchases.
type CanteenWallet struct {
	StudentID    string               `json:"student_id"`
	Balance      float64              `json:"balance"`
	Transactions []CanteenTransaction `json:"transactions"`
}

// CanteenMenuItem represents daily cafeteria menu items with pricing and tags.
type CanteenMenuItem struct {
	ID          string  `json:"id"`
	DayOfWeek   string  `json:"day"`
	Name        string  `json:"name"`
	Category    string  `json:"category"`
	DietaryTag  string  `json:"dietary_tag"` // Veg, Non-Veg, Vegan, Jain
	Price       float64 `json:"price"`
	IsAvailable bool    `json:"is_available"`
}
