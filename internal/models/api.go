package models

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// APIResponse standardizes JSON responses with audit tracking reference numbers.
type APIResponse struct {
	Success     bool        `json:"success"`
	ReferenceNo string      `json:"reference_no,omitempty"`
	Timestamp   string      `json:"timestamp"`
	Data        interface{} `json:"data,omitempty"`
	Error       string      `json:"error,omitempty"`
}

// UserClaims represents decoded JWT credentials with RBAC metadata and linked students.
type UserClaims struct {
	UserID           string   `json:"user_id"`
	Role             string   `json:"role"` // superadmin, admin, teacher, student, parent, staff
	ClassID          string   `json:"class_id,omitempty"`
	SectionID        string   `json:"section_id,omitempty"`
	LinkedStudentIDs []string `json:"linked_student_ids,omitempty"`
	jwt.RegisteredClaims
}

// HasStudentAccess guards against IDOR attacks by verifying student authorization.
func (c *UserClaims) HasStudentAccess(targetStudentID string) bool {
	if c.Role == "superadmin" || c.Role == "admin" || c.Role == "teacher" {
		return true
	}
	if c.UserID == targetStudentID {
		return true
	}
	for _, id := range c.LinkedStudentIDs {
		if id == targetStudentID {
			return true
		}
	}
	return false
}

// AuditLog represents an immutable mutation log record.
type AuditLog struct {
	ReferenceNo    string    `json:"reference_no"`
	ActorID        string    `json:"actor_id"`
	ActorRole      string    `json:"actor_role"`
	Action         string    `json:"action"`
	TargetResource string    `json:"target_resource"`
	PayloadHash    string    `json:"payload_hash"`
	IPAddress      string    `json:"ip_address"`
	Timestamp      time.Time `json:"timestamp"`
}

// --- Category 1: Student Data ---
type StudentData struct {
	StudentID        string   `json:"student_id"`
	Name             string   `json:"name"`
	ClassID          string   `json:"class_id"`
	SectionID        string   `json:"section_id"`
	RollNo           string   `json:"roll_no"`
	AdmissionID      string   `json:"admission_id"`
	DOB              string   `json:"dob"`
	FatherName       string   `json:"father_name"`
	MotherName       string   `json:"mother_name"`
	BloodGroup       string   `json:"blood_group"`
	Mobile           string   `json:"mobile"`
	Email            string   `json:"email"`
	Address          string   `json:"address"`
	ProfilePhotoURL  string   `json:"profile_photo_url"`
	MedicalNotes     string   `json:"medical_notes"`
	LinkedParentIDs  []string `json:"linked_parent_ids,omitempty"`
}

type NewStudentRequest struct {
	StudentID    string `json:"student_id"`
	Name         string `json:"name"`
	ClassID      string `json:"class_id"`
	SectionID    string `json:"section_id"`
	RollNo       string `json:"roll_no"`
	DOB          string `json:"dob"`
	FatherName   string `json:"father_name"`
	MotherName   string `json:"mother_name"`
	Mobile       string `json:"mobile"`
	Email        string `json:"email"`
	Address      string `json:"address"`
}

type UpdateStudentRequest struct {
	StudentID       string `json:"student_id"`
	Mobile          string `json:"mobile,omitempty"`
	Email           string `json:"email,omitempty"`
	Address         string `json:"address,omitempty"`
	ProfilePhotoURL string `json:"profile_photo_url,omitempty"`
	MedicalNotes    string `json:"medical_notes,omitempty"`
}

// --- Category 2: Academics ---
type TimeTablePeriod struct {
	PeriodNo  int    `json:"period_no"`
	Subject   string `json:"subject"`
	Teacher   string `json:"teacher"`
	Room      string `json:"room"`
	StartTime string `json:"start_time"`
	EndTime   string `json:"end_time"`
	IsBreak   bool   `json:"is_break"`
}

type TimeTableData struct {
	ClassID   string                     `json:"class_id"`
	SectionID string                     `json:"section_id"`
	Schedule  map[string][]TimeTablePeriod `json:"schedule"` // e.g. "Monday" -> []TimeTablePeriod
}

type UpdateTimeTableRequest struct {
	ClassID   string                     `json:"class_id"`
	SectionID string                     `json:"section_id"`
	Schedule  map[string][]TimeTablePeriod `json:"schedule"`
}

type LabSchedule struct {
	LabName   string `json:"lab_name"`
	Subject   string `json:"subject"`
	Teacher   string `json:"teacher"`
	Room      string `json:"room"`
	DayOfWeek string `json:"day_of_week"`
	Timing    string `json:"timing"`
}

type SyllabusChapter struct {
	ChapterNo   int    `json:"chapter_no"`
	Title       string `json:"title"`
	Completed   bool   `json:"completed"`
	TargetDate  string `json:"target_date"`
}

type SubjectSyllabus struct {
	Subject  string            `json:"subject"`
	Chapters []SyllabusChapter `json:"chapters"`
}

type DiaryRemark struct {
	ID        string    `json:"id"`
	ClassID   string    `json:"class_id"`
	SectionID string    `json:"section_id"`
	StudentID string    `json:"student_id,omitempty"`
	Teacher   string    `json:"teacher"`
	Remark    string    `json:"remark"`
	Conduct   string    `json:"conduct"` // Positive, Neutral, Warning
	Date      string    `json:"date"`
	CreatedAt time.Time `json:"created_at"`
}

type PostDiaryRequest struct {
	ClassID   string `json:"class_id"`
	SectionID string `json:"section_id"`
	StudentID string `json:"student_id,omitempty"`
	Remark    string `json:"remark"`
	Conduct   string `json:"conduct"`
}

type HomeworkAssignment struct {
	ID             string    `json:"id"`
	ClassID        string    `json:"class_id"`
	SectionID      string    `json:"section_id"`
	Subject        string    `json:"subject"`
	Title          string    `json:"title"`
	Description    string    `json:"description"`
	Teacher        string    `json:"teacher"`
	DueDate        string    `json:"due_date"`
	AttachmentURL  string    `json:"attachment_url,omitempty"`
	SubmittedCount int       `json:"submitted_count"`
	CreatedAt      time.Time `json:"created_at"`
}

type PostHomeworkRequest struct {
	ClassID       string `json:"class_id"`
	SectionID     string `json:"section_id"`
	Subject       string `json:"subject"`
	Title         string `json:"title"`
	Description   string `json:"description"`
	DueDate       string `json:"due_date"`
	AttachmentURL string `json:"attachment_url,omitempty"`
}

type HomeworkSubmissionEntry struct {
	HomeworkID    string    `json:"homework_id"`
	StudentID     string    `json:"student_id"`
	FileURL       string    `json:"file_url"`
	SubmittedAt   time.Time `json:"submitted_at"`
	ObtainedScore float64   `json:"obtained_score,omitempty"`
}

type BatchAttendanceRequest struct {
	ClassID   string            `json:"class_id"`
	SectionID string            `json:"section_id"`
	Date      string            `json:"date"`
	Records   map[string]string `json:"records"` // student_id -> "PRESENT"|"ABSENT"|"LATE"
}

type AttendanceReport struct {
	StudentID       string             `json:"student_id"`
	Month           int                `json:"month"`
	Year            int                `json:"year"`
	TotalDays       int                `json:"total_days"`
	PresentDays     int                `json:"present_days"`
	AbsentDays      int                `json:"absent_days"`
	LateDays        int                `json:"late_days"`
	Percentage      float64            `json:"percentage"`
	DailyRecords    []AttendanceRecord `json:"daily_records"`
}

type StudentLeaveRecord struct {
	ID            string    `json:"id"`
	StudentID     string    `json:"student_id"`
	LeaveType     string    `json:"leave_type"`
	StartDate     string    `json:"start_date"`
	EndDate       string    `json:"end_date"`
	Reason        string    `json:"reason"`
	AttachmentURL string    `json:"attachment_url,omitempty"`
	Status        string    `json:"status"` // PENDING, APPROVED, REJECTED
	ReviewRemarks string    `json:"review_remarks,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
}

type ReviewLeaveRequest struct {
	LeaveID string `json:"leave_id"`
	Status  string `json:"status"` // APPROVED, REJECTED
	Remarks string `json:"remarks"`
}

// --- Category 3: Exams & Results ---
type ExamScheduleEntry struct {
	ID        string `json:"id"`
	ClassID   string `json:"class_id"`
	ExamName  string `json:"exam_name"`
	Subject   string `json:"subject"`
	Date      string `json:"date"`
	Shift     string `json:"shift"` // Morning, Afternoon
	Room      string `json:"room"`
}

type SetExamDatesRequest struct {
	ClassID  string              `json:"class_id"`
	ExamName string              `json:"exam_name"`
	Entries  []ExamScheduleEntry `json:"entries"`
}

type ExamGuidelineData struct {
	ExamName     string   `json:"exam_name"`
	Guidelines   []string `json:"guidelines"`
	SeatNumber   string   `json:"seat_number"`
	AdmitCardURL string   `json:"admit_card_url"`
}

type ExamScoreRecord struct {
	StudentID  string  `json:"student_id"`
	ExamName   string  `json:"exam_name"`
	Subject    string  `json:"subject"`
	MarksScored float64 `json:"marks_scored"`
	TotalMarks  float64 `json:"total_marks"`
	Percentile float64 `json:"percentile"`
	Grade      string  `json:"grade"`
	Comments   string  `json:"comments"`
}

type BatchSaveResultsRequest struct {
	ClassID  string            `json:"class_id"`
	ExamName string            `json:"exam_name"`
	Subject  string            `json:"subject"`
	Scores   []ExamScoreRecord `json:"scores"`
}

// --- Category 4: Payments & GST ---
type CreatePaymentOrderRequest struct {
	StudentID   string  `json:"student_id"`
	Amount      float64 `json:"amount"` // in INR
	FeeType     string  `json:"fee_type"` // Tuition, Transport, Lab, Canteen
	Description string  `json:"description"`
}

type PaymentOrderData struct {
	OrderID     string  `json:"order_id"`
	Amount      float64 `json:"amount"`
	Currency    string  `json:"currency"`
	KeyID       string  `json:"key_id"`
	ReceiptID   string  `json:"receipt_id"`
}

type VerifyPaymentRequest struct {
	OrderID   string `json:"order_id"`
	PaymentID string `json:"payment_id"`
	Signature string `json:"signature"`
	StudentID string `json:"student_id"`
}

type StudentPaymentTransaction struct {
	ID            string    `json:"id"`
	StudentID     string    `json:"student_id"`
	OrderID       string    `json:"order_id"`
	PaymentID     string    `json:"payment_id"`
	Amount        float64   `json:"amount"`
	Currency      string    `json:"currency"`
	Status        string    `json:"status"` // SUCCESS, FAILED
	Purpose       string    `json:"purpose"`
	ReceiptRef    string    `json:"receipt_ref"`
	CreatedAt     time.Time `json:"created_at"`
}

type GenerateTaxReceiptRequest struct {
	StudentID string  `json:"student_id"`
	PaymentID string  `json:"payment_id"`
	GSTIN     string  `json:"gstin"`
	StateCode string  `json:"state_code"`
}

type TaxReceiptData struct {
	ReceiptNo     string    `json:"receipt_no"`
	GSTIN         string    `json:"gstin"`
	SchoolGSTIN   string    `json:"school_gstin"`
	TaxableAmount float64   `json:"taxable_amount"`
	CGST          float64   `json:"cgst"`
	SGST          float64   `json:"sgst"`
	IGST          float64   `json:"igst"`
	TotalAmount   float64   `json:"total_amount"`
	Date          string    `json:"date"`
}

// --- Category 5: Logistics, Health & Store ---
type BusTravelInfo struct {
	RouteNumber string          `json:"route_number"`
	DriverName  string          `json:"driver_name"`
	DriverPhone string          `json:"driver_phone"`
	PickupTime  string          `json:"pickup_time"`
	DropTime    string          `json:"drop_time"`
	CurrentLat  float64         `json:"current_lat"`
	CurrentLng  float64         `json:"current_lng"`
	Stops       []TransportStop `json:"stops"`
}

type CanteenOrderRequest struct {
	StudentID string                   `json:"student_id"`
	Items     []modelsCanteenOrderItem `json:"items"`
}

type modelsCanteenOrderItem struct {
	ItemID   string  `json:"item_id"`
	Name     string  `json:"name"`
	Price    float64 `json:"price"`
	Quantity int     `json:"quantity"`
}

type CanteenMealOrder struct {
	ID          string                   `json:"id"`
	StudentID   string                   `json:"student_id"`
	Items       []modelsCanteenOrderItem `json:"items"`
	TotalAmount float64                  `json:"total_amount"`
	TokenNumber string                   `json:"token_number"`
	Status      string                   `json:"status"` // PREPARING, READY, COLLECTED
	CreatedAt   time.Time                `json:"created_at"`
}

type IssueHallPassRequest struct {
	StudentID       string `json:"student_id"`
	Destination     string `json:"destination"`
	DurationMinutes int    `json:"duration_minutes"`
}

type UpdateHealthRequest struct {
	StudentID         string   `json:"student_id"`
	BloodGroup        string   `json:"blood_group,omitempty"`
	Allergies         []string `json:"allergies,omitempty"`
	ChronicConditions []string `json:"chronic_conditions,omitempty"`
	InfirmaryVisit    string   `json:"infirmary_visit,omitempty"`
}

type CartItem struct {
	ProductID string  `json:"product_id"`
	Title     string  `json:"title"`
	Size      string  `json:"size,omitempty"`
	Price     float64 `json:"price"`
	Quantity  int     `json:"quantity"`
}

type UpdateStoreCartRequest struct {
	StudentID string     `json:"student_id"`
	Items     []CartItem `json:"items"`
}

// --- Category 6: Campus Life & Media ---
type AddHousePointsRequest struct {
	HouseName string `json:"house_name"`
	Points    int    `json:"points"`
	Reason    string `json:"reason"`
	StudentID string `json:"student_id,omitempty"`
}

type ClaimLostItemRequest struct {
	ItemID    string `json:"item_id"`
	StudentID string `json:"student_id"`
	ProofDesc string `json:"proof_desc"`
}

// --- Category 7: Parent Desk, PTM & Notifications ---
type ParentMessageThread struct {
	ID          string          `json:"id"`
	StudentID   string          `json:"student_id"`
	ParentID    string          `json:"parent_id"`
	TeacherID   string          `json:"teacher_id"`
	Subject     string          `json:"subject"`
	LastMessage string          `json:"last_message"`
	Messages    []ThreadMessage `json:"messages"`
	UpdatedAt   time.Time       `json:"updated_at"`
}

type ThreadMessage struct {
	SenderID   string    `json:"sender_id"`
	SenderRole string    `json:"sender_role"`
	Content    string    `json:"content"`
	Timestamp  time.Time `json:"timestamp"`
}

type PostParentDeskRequest struct {
	ThreadID  string `json:"thread_id,omitempty"`
	StudentID string `json:"student_id"`
	TeacherID string `json:"teacher_id"`
	Subject   string `json:"subject"`
	Message   string `json:"message"`
}

type PtmTeacherTimingData struct {
	TeacherID   string   `json:"teacher_id"`
	TeacherName string   `json:"teacher_name"`
	Subject     string   `json:"subject"`
	Slots       []string `json:"slots"`
}

type PtmQuickBookReq struct {
	StudentID string `json:"student_id"`
	TeacherID string `json:"teacher_id"`
	SlotTime  string `json:"slot_time"`
	Agenda    string `json:"agenda"`
}

type ReschedulePtmReq struct {
	BookingID   string `json:"booking_id"`
	NewSlotTime string `json:"new_slot_time"`
}

type AddCalendarEventRequest struct {
	Title     string `json:"title"`
	StartTime string `json:"start_time"`
	EndTime   string `json:"end_time"`
	Location  string `json:"location"`
	Details   string `json:"details"`
}

type UrgentNoticeBroadcastRequest struct {
	Title         string   `json:"title"`
	Message       string   `json:"message"`
	Priority      string   `json:"priority"` // URGENT, EMERGENCY
	TargetClasses []string `json:"target_classes,omitempty"`
	SendFCM       bool     `json:"send_fcm"`
	SendSMS       bool     `json:"send_sms"`
}

type PostNoticeRequest struct {
	Title          string `json:"title"`
	Body           string `json:"body"`
	Category       string `json:"category"`
	TargetClass    string `json:"target_class,omitempty"`
	AttachmentName string `json:"attachment_name,omitempty"`
}

type NotificationTrackingReport struct {
	ReferenceID string  `json:"reference_id"`
	Title       string  `json:"title"`
	SentCount   int     `json:"sent_count"`
	Delivered   int     `json:"delivered"`
	ReadCount   int     `json:"read_count"`
	DeliveryPct float64 `json:"delivery_pct"`
}

// --- Category 8: Staff & System ---
type SchoolInfoData struct {
	Name         string   `json:"name"`
	Affiliation  string   `json:"affiliation"`
	SchoolCode   string   `json:"school_code"`
	Address      string   `json:"address"`
	Phone        string   `json:"phone"`
	Email        string   `json:"email"`
	WorkingHours string   `json:"working_hours"`
	TermDates    []string `json:"term_dates"`
}

type AddStaffRequest struct {
	Name            string   `json:"name"`
	Email           string   `json:"email"`
	Phone           string   `json:"phone"`
	Role            string   `json:"role"` // teacher, staff, coordinator
	Designation     string   `json:"designation"`
	AssignedClasses []string `json:"assigned_classes"`
}

type UpdateStaffRequest struct {
	StaffID         string   `json:"staff_id"`
	Designation     string   `json:"designation,omitempty"`
	AssignedClasses []string `json:"assigned_classes,omitempty"`
	Phone           string   `json:"phone,omitempty"`
	Role            string   `json:"role,omitempty"`
}

type HealthCheckMetrics struct {
	Status           string  `json:"status"`
	UptimeSeconds    float64 `json:"uptime_seconds"`
	GCMemoryMB       float64 `json:"gc_memory_mb"`
	AllocatedMB      float64 `json:"allocated_mb"`
	TotalAllocMB     float64 `json:"total_alloc_mb"`
	ActiveGoroutines int     `json:"active_goroutines"`
	StorageMode      string  `json:"storage_mode"`
	DatabasePing     string  `json:"database_ping"`
	CacheItems       int     `json:"cache_items"`
}
