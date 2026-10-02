package models

import (
	"time"
)

// AttendanceRecord represents a single day's attendance status for a student.
type AttendanceRecord struct {
	ID         string `json:"id,omitempty"`
	StudentID  string `json:"student_id,omitempty"`
	Date       string `json:"date"`
	DateIso    string `json:"date_iso,omitempty"`
	DayOfMonth int    `json:"day_of_month"`
	Status     string `json:"status"` // PRESENT, ABSENT, HOLIDAY, SUNDAY, LATE, NOT_MARKED
	Remarks    string `json:"remarks,omitempty"`
}

// AttendanceMonthResponse encapsulates the calendar matrix and monthly summary.
type AttendanceMonthResponse struct {
	StudentID     string             `json:"student_id"`
	Month         int                `json:"month"`
	Year          int                `json:"year"`
	TotalDays     int                `json:"total_days"`
	PresentDays   int                `json:"present_days"`
	AbsentDays    int                `json:"absent_days"`
	HolidayDays   int                `json:"holiday_days"`
	AttendancePct float64            `json:"attendance_pct"`
	Records       []AttendanceRecord `json:"records"`
}

// AttendanceSummaryResponse represents overall academic term metrics for GET /v1/attendance/summary
type AttendanceSummaryResponse struct {
	StudentID         string  `json:"student_id"`
	TotalWorkingDays  int     `json:"total_working_days"`
	DaysPresent       int     `json:"days_present"`
	DaysAbsent        int     `json:"days_absent"`
	DaysLate          int     `json:"days_late"`
	Holidays          int     `json:"holidays"`
	OverallPercentage float64 `json:"overall_percentage"`
}

// LeaveApplicationRequest is the payload for POST /v1/attendance/leave
type LeaveApplicationRequest struct {
	StudentID     string `json:"student_id"`
	LeaveType     string `json:"leave_type"`
	StartDate     string `json:"start_date"`
	EndDate       string `json:"end_date"`
	Reason        string `json:"reason"`
	AttachmentURL string `json:"attachment_url,omitempty"`
}

// LeaveApplication represents a submitted leave request.
type LeaveApplication struct {
	ID            string    `json:"id"`
	StudentID     string    `json:"student_id"`
	LeaveType     string    `json:"leave_type"`
	StartDate     string    `json:"start_date"`
	EndDate       string    `json:"end_date"`
	Reason        string    `json:"reason"`
	AttachmentURL *string   `json:"attachment_url,omitempty"`
	Status        string    `json:"status"` // pending, approved, rejected
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at,omitempty"`
}
