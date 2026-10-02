package models

// FeeItem represents a fee component or installment.
type FeeItem struct {
	ID          string  `json:"id"`
	StudentID   string  `json:"student_id,omitempty"`
	Title       string  `json:"title"`
	Subtitle    string  `json:"subtitle"`
	Amount      int     `json:"amount"`
	DueDate     string  `json:"due_date,omitempty"`
	Checked     bool    `json:"checked"`
	Cleared     bool    `json:"cleared"`
	ClearedMeta *string `json:"cleared_meta,omitempty"`
}

// FeeHistory represents a completed fee payment transaction record.
type FeeHistory struct {
	ID        string `json:"id,omitempty"`
	StudentID string `json:"student_id,omitempty"`
	Title     string `json:"title"`
	Date      string `json:"date"`
	Amount    int    `json:"amount"`
	Ref       string `json:"ref"`
}

// FeeSummary aggregates pending fee items, payment history, and balances.
type FeeSummary struct {
	StudentID  string       `json:"student_id"`
	TotalDue   int          `json:"total_due"`
	TotalPaid  int          `json:"total_paid"`
	DueDate    string       `json:"fee_due_date"`
	Currency   string       `json:"currency"`
	Items      []FeeItem    `json:"items"`
	History    []FeeHistory `json:"history"`
}
