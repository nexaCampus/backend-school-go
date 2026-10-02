package models

import "time"

// --- 7. Academic Performance & Report Card ---
type SubjectGrade struct {
	Subject string  `json:"subject"`
	Score   float64 `json:"score"`
	Total   float64 `json:"total"`
	Grade   string  `json:"grade"`
	Remarks string  `json:"remarks"`
}

type ReportCard struct {
	ID             string         `json:"id"`
	StudentID      string         `json:"student_id"`
	StudentName    string         `json:"student_name"`
	Term           string         `json:"term"`
	AcademicYear   string         `json:"academic_year"`
	GPA            float64        `json:"gpa"`
	OverallGrade   string         `json:"overall_grade"`
	TotalScored    float64        `json:"total_scored"`
	TotalMax       float64        `json:"total_max"`
	Rank           int            `json:"rank"`
	TeacherRemarks string         `json:"teacher_remarks"`
	MarksheetURL   string         `json:"marksheet_url"`
	Subjects       []SubjectGrade `json:"subjects"`
}

// --- 14. Parent-Teacher Meeting (PTM) ---
type PtmSession struct {
	ID          string `json:"id"`
	Grade       string `json:"grade"`
	Title       string `json:"title"`
	Date        string `json:"date"`
	TimeRange   string `json:"time_range"`
	FacultyName string `json:"faculty_name"`
	Location    string `json:"location"`
}

type PtmSlot struct {
	ID          string `json:"id"`
	TeacherID   string `json:"teacher_id"`
	TeacherName string `json:"teacher_name"`
	Date        string `json:"date"`
	TimeSlot    string `json:"time_slot"`
	IsBooked    bool   `json:"is_booked"`
}

type PtmBookingRequest struct {
	StudentID string `json:"student_id"`
	SlotID    string `json:"slot_id"`
	Agenda    string `json:"agenda"`
}

type PtmBooking struct {
	ID          string    `json:"id"`
	StudentID   string    `json:"student_id"`
	SlotID      string    `json:"slot_id"`
	TeacherName string    `json:"teacher_name"`
	Date        string    `json:"date"`
	TimeSlot    string    `json:"time_slot"`
	Agenda      string    `json:"agenda"`
	MeetingLink string    `json:"meeting_link"`
	Status      string    `json:"status"` // CONFIRMED, CANCELLED
	CreatedAt   time.Time `json:"created_at"`
}

// --- 15. Digital Hall Pass ---
type HallPassRequest struct {
	StudentID       string `json:"student_id"`
	Destination     string `json:"destination"`
	DurationMinutes int    `json:"duration_minutes"`
}

type HallPass struct {
	ID               string    `json:"id"`
	StudentID        string    `json:"student_id"`
	Destination      string    `json:"destination"`
	DurationMinutes  int       `json:"duration_minutes"`
	TeacherName      string    `json:"teacher_name"`
	Status           string    `json:"status"` // ACTIVE, EXPIRED, RETURNED
	VerificationQR   string    `json:"verification_qr"`
	IssuedAt         time.Time `json:"issued_at"`
	ExpiresAt        time.Time `json:"expires_at"`
	RemainingSeconds int       `json:"remaining_seconds"`
}

// --- 16. Student Infirmary & Health ---
type HealthProfile struct {
	StudentID             string   `json:"student_id"`
	BloodGroup            string   `json:"blood_group"`
	Allergies             []string `json:"allergies"`
	Conditions            []string `json:"conditions"`
	EmergencyContactName  string   `json:"emergency_contact_name"`
	EmergencyContactPhone string   `json:"emergency_contact_phone"`
}

type ClinicVisit struct {
	ID          string `json:"id"`
	StudentID   string `json:"student_id"`
	Date        string `json:"date"`
	Temperature string `json:"temperature"`
	Complaint   string `json:"complaint"`
	Treatment   string `json:"treatment"`
	FirstAid    string `json:"first_aid"`
	NurseName   string `json:"nurse_name"`
}

type HealthConsentRequest struct {
	StudentID       string `json:"student_id"`
	MedicationName  string `json:"medication_name"`
	PrescriptionURL string `json:"prescription_url"`
}

type HealthConsent struct {
	ID              string    `json:"id"`
	StudentID       string    `json:"student_id"`
	MedicationName  string    `json:"medication_name"`
	PrescriptionURL string    `json:"prescription_url"`
	Status          string    `json:"status"` // AUTHORIZED, PENDING
	CreatedAt       time.Time `json:"created_at"`
}

// --- 17. House System, Merits & Discipline ---
type HouseStanding struct {
	HouseName   string `json:"house_name"`
	HouseColor  string `json:"house_color"`
	TotalPoints int    `json:"total_points"`
	Rank        int    `json:"rank"`
}

type MeritSummary struct {
	StudentID        string          `json:"student_id"`
	HouseName        string          `json:"house_name"`
	HouseColor       string          `json:"house_color"`
	IndividualPoints int             `json:"individual_points"`
	Leaderboard      []HouseStanding `json:"leaderboard"`
}

type MeritRecord struct {
	ID        string `json:"id"`
	StudentID string `json:"student_id"`
	Type      string `json:"type"` // MERIT, DISCIPLINE
	Title     string `json:"title"`
	Points    int    `json:"points"`
	Remarks   string `json:"remarks"`
	AwardedBy string `json:"awarded_by"`
	Date      string `json:"date"`
}

// --- 18. School Store & Inventory ---
type StoreProduct struct {
	ID            string   `json:"id"`
	Title         string   `json:"title"`
	Category      string   `json:"category"`
	Price         float64  `json:"price"`
	Sizes         []string `json:"sizes"`
	ImageURL      string   `json:"image_url"`
	InStock       bool     `json:"in_stock"`
	StockQuantity int      `json:"stock_quantity"`
}

type StoreOrderItem struct {
	ProductID string  `json:"product_id"`
	Title     string  `json:"title"`
	Size      string  `json:"size,omitempty"`
	Quantity  int     `json:"quantity"`
	UnitPrice float64 `json:"unit_price"`
}

type StoreOrderRequest struct {
	StudentID      string           `json:"student_id"`
	Items          []StoreOrderItem `json:"items"`
	DeliveryOption string           `json:"delivery_option"` // locker, desk
}

type StoreOrder struct {
	ID             string           `json:"id"`
	StudentID      string           `json:"student_id"`
	Items          []StoreOrderItem `json:"items"`
	TotalAmount    float64          `json:"total_amount"`
	DeliveryOption string           `json:"delivery_option"`
	Status         string           `json:"status"` // PRE_ORDERED, DELIVERED
	CreatedAt      time.Time        `json:"created_at"`
}

// --- 19. Campus Lost & Found ---
type LostFoundItem struct {
	ID               string    `json:"id"`
	StudentID        string    `json:"student_id,omitempty"`
	ItemName         string    `json:"item_name"`
	LastSeenLocation string    `json:"last_seen_location"`
	ImageURL         *string   `json:"image_url,omitempty"`
	Status           string    `json:"status"` // found, claimed
	CreatedAt        time.Time `json:"created_at"`
}

type ReportLostItemRequest struct {
	StudentID        string `json:"student_id"`
	ItemName         string `json:"item_name"`
	LastSeenLocation string `json:"last_seen_location"`
	ImageURL         string `json:"image_url,omitempty"`
}

// --- 20. Event Gallery & Yearbook ---
type GalleryAlbum struct {
	ID         string `json:"id"`
	Title      string `json:"title"`
	Year       string `json:"year"`
	CoverURL   string `json:"cover_url"`
	PhotoCount int    `json:"photo_count"`
	Date       string `json:"date"`
}

type GalleryPhoto struct {
	ID      string `json:"id"`
	AlbumID string `json:"album_id"`
	URL     string `json:"url"`
	Caption string `json:"caption"`
}

// --- 21. Clubs & Societies ---
type EnrolledClub struct {
	ID              string `json:"id"`
	Name            string `json:"name"`
	Description     string `json:"description"`
	Role            string `json:"role"` // MEMBER, LEAD
	FacultyAdvisor  string `json:"faculty_advisor"`
	MeetingSchedule string `json:"meeting_schedule"`
	BannerURL       string `json:"banner_url"`
}

type Competition struct {
	ID               string `json:"id"`
	Title            string `json:"title"`
	Grade            string `json:"grade"`
	Description      string `json:"description"`
	Deadline         string `json:"deadline"`
	RegistrationLink string `json:"registration_link"`
}

type CompetitionRegisterRequest struct {
	StudentID string `json:"student_id"`
}

type CompetitionRegistration struct {
	ID            string    `json:"id"`
	CompetitionID string    `json:"competition_id"`
	StudentID     string    `json:"student_id"`
	Status        string    `json:"status"`
	RegisteredAt  time.Time `json:"registered_at"`
}
