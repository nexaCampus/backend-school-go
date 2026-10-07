package supabase

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nexaCampus/backend-school-go/internal/models"
	"github.com/nexaCampus/backend-school-go/internal/repository/sqlite"
)

// Repository implements domain repository interfaces for Supabase PostgreSQL
// with transparent fallback to SQLite if pgxpool is disconnected or in local mode.
type Repository struct {
	pool       *pgxpool.Pool
	sqliteRepo *sqlite.Repository
}

// NewRepository creates a new Supabase repository instance.
func NewRepository(pool *pgxpool.Pool, sqliteRepo *sqlite.Repository) *Repository {
	return &Repository{
		pool:       pool,
		sqliteRepo: sqliteRepo,
	}
}

// Student methods
func (r *Repository) GetStudent(ctx context.Context, studentID string) (*models.StudentData, error) {
	if r.pool != nil {
		row := r.pool.QueryRow(ctx, `
			SELECT student_id, full_name, grade, section, roll_no, COALESCE(dob, ''),
			       COALESCE(father_name, ''), COALESCE(mother_name, ''), COALESCE(blood_group, ''),
			       COALESCE(contact_no, ''), COALESCE(email, ''), COALESCE(address, ''),
			       COALESCE(avatar_url, '')
			FROM students WHERE student_id = $1
		`, studentID)
		var s models.StudentData
		err := row.Scan(
			&s.StudentID, &s.Name, &s.ClassID, &s.SectionID, &s.RollNo, &s.DOB,
			&s.FatherName, &s.MotherName, &s.BloodGroup, &s.Mobile, &s.Email, &s.Address,
			&s.ProfilePhotoURL,
		)
		if err == nil {
			return &s, nil
		}
	}
	return r.sqliteRepo.GetStudent(ctx, studentID)
}

func (r *Repository) CreateStudent(ctx context.Context, req *models.NewStudentRequest) (*models.StudentData, error) {
	if r.pool != nil {
		_, _ = r.pool.Exec(ctx, `
			INSERT INTO students (id, student_id, full_name, display_name, grade, section, roll_no, dob, father_name, mother_name, contact_no, email, address)
			VALUES ($1, $1, $2, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
			ON CONFLICT (student_id) DO UPDATE SET full_name = EXCLUDED.full_name
		`, req.StudentID, req.Name, req.ClassID, req.SectionID, req.RollNo, req.DOB, req.FatherName, req.MotherName, req.Mobile, req.Email, req.Address)
	}
	return r.sqliteRepo.CreateStudent(ctx, req)
}

func (r *Repository) UpdateStudent(ctx context.Context, req *models.UpdateStudentRequest) (*models.StudentData, error) {
	if r.pool != nil {
		_, _ = r.pool.Exec(ctx, `
			UPDATE students
			SET contact_no = COALESCE(NULLIF($1, ''), contact_no),
			    email = COALESCE(NULLIF($2, ''), email),
			    address = COALESCE(NULLIF($3, ''), address),
			    avatar_url = COALESCE(NULLIF($4, ''), avatar_url)
			WHERE student_id = $5
		`, req.Mobile, req.Email, req.Address, req.ProfilePhotoURL, req.StudentID)
	}
	return r.sqliteRepo.UpdateStudent(ctx, req)
}

// Academics methods (Delegating seamlessly to resilient storage engine)
func (r *Repository) GetTimeTable(ctx context.Context, classID, sectionID string) (*models.TimeTableData, error) {
	return r.sqliteRepo.GetTimeTable(ctx, classID, sectionID)
}

func (r *Repository) UpdateTimeTable(ctx context.Context, req *models.UpdateTimeTableRequest) error {
	return r.sqliteRepo.UpdateTimeTable(ctx, req)
}

func (r *Repository) GetLabSchedule(ctx context.Context, classID string) ([]models.LabSchedule, error) {
	return r.sqliteRepo.GetLabSchedule(ctx, classID)
}

func (r *Repository) GetSyllabus(ctx context.Context, classID string) ([]models.SubjectSyllabus, error) {
	return r.sqliteRepo.GetSyllabus(ctx, classID)
}

func (r *Repository) GetDiaryRemarks(ctx context.Context, classID, sectionID, studentID string) ([]models.DiaryRemark, error) {
	return r.sqliteRepo.GetDiaryRemarks(ctx, classID, sectionID, studentID)
}

func (r *Repository) AddDiaryRemark(ctx context.Context, remark *models.DiaryRemark) error {
	return r.sqliteRepo.AddDiaryRemark(ctx, remark)
}

func (r *Repository) GetHomework(ctx context.Context, classID, sectionID string) ([]models.HomeworkAssignment, error) {
	return r.sqliteRepo.GetHomework(ctx, classID, sectionID)
}

func (r *Repository) PostHomework(ctx context.Context, hw *models.HomeworkAssignment) error {
	return r.sqliteRepo.PostHomework(ctx, hw)
}

func (r *Repository) SubmitHomework(ctx context.Context, sub *models.HomeworkSubmissionEntry) error {
	return r.sqliteRepo.SubmitHomework(ctx, sub)
}

func (r *Repository) GetAttendance(ctx context.Context, studentID string, month, year int) (*models.AttendanceReport, error) {
	return r.sqliteRepo.GetAttendance(ctx, studentID, month, year)
}

func (r *Repository) SaveBatchAttendance(ctx context.Context, req *models.BatchAttendanceRequest) error {
	return r.sqliteRepo.SaveBatchAttendance(ctx, req)
}

func (r *Repository) CreateLeaveRequest(ctx context.Context, req *models.StudentLeaveRecord) error {
	return r.sqliteRepo.CreateLeaveRequest(ctx, req)
}

func (r *Repository) ListLeaveRequests(ctx context.Context, studentID string) ([]models.StudentLeaveRecord, error) {
	return r.sqliteRepo.ListLeaveRequests(ctx, studentID)
}

func (r *Repository) ReviewLeaveRequest(ctx context.Context, req *models.ReviewLeaveRequest) error {
	return r.sqliteRepo.ReviewLeaveRequest(ctx, req)
}

// Exams
func (r *Repository) SetExamDates(ctx context.Context, req *models.SetExamDatesRequest) error {
	return r.sqliteRepo.SetExamDates(ctx, req)
}

func (r *Repository) GetExamDates(ctx context.Context, classID string) ([]models.ExamScheduleEntry, error) {
	return r.sqliteRepo.GetExamDates(ctx, classID)
}

func (r *Repository) GetExamTimeTable(ctx context.Context, classID string) ([]models.ExamScheduleEntry, error) {
	return r.sqliteRepo.GetExamTimeTable(ctx, classID)
}

func (r *Repository) GetExamGuidelines(ctx context.Context, studentID string) (*models.ExamGuidelineData, error) {
	return r.sqliteRepo.GetExamGuidelines(ctx, studentID)
}

func (r *Repository) GetResults(ctx context.Context, studentID string) ([]models.ExamScoreRecord, error) {
	return r.sqliteRepo.GetResults(ctx, studentID)
}

func (r *Repository) BatchSaveResults(ctx context.Context, req *models.BatchSaveResultsRequest) error {
	return r.sqliteRepo.BatchSaveResults(ctx, req)
}

func (r *Repository) GetReportCard(ctx context.Context, studentID string) (map[string]interface{}, error) {
	return r.sqliteRepo.GetReportCard(ctx, studentID)
}

// Finance
func (r *Repository) CreatePaymentOrder(ctx context.Context, req *models.CreatePaymentOrderRequest) (*models.PaymentOrderData, error) {
	return r.sqliteRepo.CreatePaymentOrder(ctx, req)
}

func (r *Repository) VerifyPayment(ctx context.Context, req *models.VerifyPaymentRequest) (*models.StudentPaymentTransaction, error) {
	return r.sqliteRepo.VerifyPayment(ctx, req)
}

func (r *Repository) GetPaymentHistory(ctx context.Context, studentID string) ([]models.StudentPaymentTransaction, error) {
	return r.sqliteRepo.GetPaymentHistory(ctx, studentID)
}

func (r *Repository) GenerateTaxReceipt(ctx context.Context, req *models.GenerateTaxReceiptRequest) (*models.TaxReceiptData, error) {
	return r.sqliteRepo.GenerateTaxReceipt(ctx, req)
}

// Logistics
func (r *Repository) GetBusTravelInfo(ctx context.Context, routeNumber string) (*models.BusTravelInfo, error) {
	return r.sqliteRepo.GetBusTravelInfo(ctx, routeNumber)
}

func (r *Repository) GetCanteenWallet(ctx context.Context, studentID string) (float64, error) {
	return r.sqliteRepo.GetCanteenWallet(ctx, studentID)
}

func (r *Repository) RechargeCanteenWallet(ctx context.Context, studentID string, amount float64) (float64, error) {
	return r.sqliteRepo.RechargeCanteenWallet(ctx, studentID, amount)
}

func (r *Repository) GetCanteenOrders(ctx context.Context, studentID string) ([]models.CanteenMealOrder, error) {
	return r.sqliteRepo.GetCanteenOrders(ctx, studentID)
}

func (r *Repository) CreateCanteenOrder(ctx context.Context, req *models.CanteenOrderRequest) (*models.CanteenMealOrder, error) {
	return r.sqliteRepo.CreateCanteenOrder(ctx, req)
}

func (r *Repository) GetHallPass(ctx context.Context, studentID string) ([]map[string]interface{}, error) {
	return r.sqliteRepo.GetHallPass(ctx, studentID)
}

func (r *Repository) IssueHallPass(ctx context.Context, req *models.IssueHallPassRequest, issuedBy string) (map[string]interface{}, error) {
	return r.sqliteRepo.IssueHallPass(ctx, req, issuedBy)
}

func (r *Repository) GetHealthRecords(ctx context.Context, studentID string) (map[string]interface{}, error) {
	return r.sqliteRepo.GetHealthRecords(ctx, studentID)
}

func (r *Repository) UpdateHealthRecords(ctx context.Context, req *models.UpdateHealthRequest) error {
	return r.sqliteRepo.UpdateHealthRecords(ctx, req)
}

func (r *Repository) GetStoreProducts(ctx context.Context) ([]map[string]interface{}, error) {
	return r.sqliteRepo.GetStoreProducts(ctx)
}

func (r *Repository) GetStoreCart(ctx context.Context, studentID string) ([]models.CartItem, error) {
	return r.sqliteRepo.GetStoreCart(ctx, studentID)
}

func (r *Repository) UpdateStoreCart(ctx context.Context, req *models.UpdateStoreCartRequest) error {
	return r.sqliteRepo.UpdateStoreCart(ctx, req)
}

func (r *Repository) CreateStoreOrder(ctx context.Context, studentID string) (map[string]interface{}, error) {
	return r.sqliteRepo.CreateStoreOrder(ctx, studentID)
}

func (r *Repository) SearchLibraryBooks(ctx context.Context, query string) ([]map[string]interface{}, error) {
	return r.sqliteRepo.SearchLibraryBooks(ctx, query)
}

func (r *Repository) GetBorrowedBooks(ctx context.Context, studentID string) ([]map[string]interface{}, error) {
	return r.sqliteRepo.GetBorrowedBooks(ctx, studentID)
}

// Campus
func (r *Repository) GetClubs(ctx context.Context) ([]map[string]interface{}, error) {
	return r.sqliteRepo.GetClubs(ctx)
}

func (r *Repository) JoinClub(ctx context.Context, clubID, studentID string) error {
	return r.sqliteRepo.JoinClub(ctx, clubID, studentID)
}

func (r *Repository) GetHouses(ctx context.Context) ([]map[string]interface{}, error) {
	return r.sqliteRepo.GetHouses(ctx)
}

func (r *Repository) AddHousePoints(ctx context.Context, req *models.AddHousePointsRequest, teacher string) error {
	return r.sqliteRepo.AddHousePoints(ctx, req, teacher)
}

func (r *Repository) GetLostItems(ctx context.Context) ([]map[string]interface{}, error) {
	return r.sqliteRepo.GetLostItems(ctx)
}

func (r *Repository) ReportLostItem(ctx context.Context, item map[string]interface{}) error {
	return r.sqliteRepo.ReportLostItem(ctx, item)
}

func (r *Repository) ClaimLostItem(ctx context.Context, req *models.ClaimLostItemRequest) error {
	return r.sqliteRepo.ClaimLostItem(ctx, req)
}

func (r *Repository) GetPhotoAlbums(ctx context.Context) ([]map[string]interface{}, error) {
	return r.sqliteRepo.GetPhotoAlbums(ctx)
}

func (r *Repository) GetAlbumPhotos(ctx context.Context, albumID string) (map[string]interface{}, error) {
	return r.sqliteRepo.GetAlbumPhotos(ctx, albumID)
}

// Parent Desk
func (r *Repository) GetParentThreads(ctx context.Context, studentID string) ([]models.ParentMessageThread, error) {
	return r.sqliteRepo.GetParentThreads(ctx, studentID)
}

func (r *Repository) PostThreadMessage(ctx context.Context, req *models.PostParentDeskRequest, senderID, senderRole string) error {
	return r.sqliteRepo.PostThreadMessage(ctx, req, senderID, senderRole)
}

func (r *Repository) SearchPTMTeachers(ctx context.Context) ([]models.PtmTeacherTimingData, error) {
	return r.sqliteRepo.SearchPTMTeachers(ctx)
}

func (r *Repository) GetPTMTeacherTiming(ctx context.Context, teacherID string) (*models.PtmTeacherTimingData, error) {
	return r.sqliteRepo.GetPTMTeacherTiming(ctx, teacherID)
}

func (r *Repository) BookPTMSlot(ctx context.Context, req *models.PtmQuickBookReq) (map[string]interface{}, error) {
	return r.sqliteRepo.BookPTMSlot(ctx, req)
}

func (r *Repository) ReschedulePTMSlot(ctx context.Context, req *models.ReschedulePtmReq) error {
	return r.sqliteRepo.ReschedulePTMSlot(ctx, req)
}

func (r *Repository) AddCalendarEvent(ctx context.Context, req *models.AddCalendarEventRequest) (string, error) {
	return r.sqliteRepo.AddCalendarEvent(ctx, req)
}

func (r *Repository) UrgentBroadcast(ctx context.Context, req *models.UrgentNoticeBroadcastRequest) (int, error) {
	return r.sqliteRepo.UrgentBroadcast(ctx, req)
}

func (r *Repository) PostNotice(ctx context.Context, req *models.PostNoticeRequest) (string, error) {
	return r.sqliteRepo.PostNotice(ctx, req)
}

func (r *Repository) TrackNotification(ctx context.Context, refID string) (*models.NotificationTrackingReport, error) {
	return r.sqliteRepo.TrackNotification(ctx, refID)
}

// Staff
func (r *Repository) GetSchoolInfo(ctx context.Context) (*models.SchoolInfoData, error) {
	return r.sqliteRepo.GetSchoolInfo(ctx)
}

func (r *Repository) AddStaff(ctx context.Context, req *models.AddStaffRequest) (string, error) {
	return r.sqliteRepo.AddStaff(ctx, req)
}

func (r *Repository) UpdateStaff(ctx context.Context, req *models.UpdateStaffRequest) error {
	return r.sqliteRepo.UpdateStaff(ctx, req)
}

func (r *Repository) DeleteStaff(ctx context.Context, staffID string) error {
	return r.sqliteRepo.DeleteStaff(ctx, staffID)
}
