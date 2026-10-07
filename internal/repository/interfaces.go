package repository

import (
	"context"

	"github.com/nexaCampus/backend-school-go/internal/models"
)

// StudentRepository defines operations for Category 1.
type StudentRepository interface {
	GetStudent(ctx context.Context, studentID string) (*models.StudentData, error)
	CreateStudent(ctx context.Context, req *models.NewStudentRequest) (*models.StudentData, error)
	UpdateStudent(ctx context.Context, req *models.UpdateStudentRequest) (*models.StudentData, error)
}

// AcademicsRepository defines operations for Category 2.
type AcademicsRepository interface {
	GetTimeTable(ctx context.Context, classID, sectionID string) (*models.TimeTableData, error)
	UpdateTimeTable(ctx context.Context, req *models.UpdateTimeTableRequest) error
	GetLabSchedule(ctx context.Context, classID string) ([]models.LabSchedule, error)
	GetSyllabus(ctx context.Context, classID string) ([]models.SubjectSyllabus, error)
	GetDiaryRemarks(ctx context.Context, classID, sectionID, studentID string) ([]models.DiaryRemark, error)
	AddDiaryRemark(ctx context.Context, remark *models.DiaryRemark) error
	GetHomework(ctx context.Context, classID, sectionID string) ([]models.HomeworkAssignment, error)
	PostHomework(ctx context.Context, hw *models.HomeworkAssignment) error
	SubmitHomework(ctx context.Context, sub *models.HomeworkSubmissionEntry) error
	GetAttendance(ctx context.Context, studentID string, month, year int) (*models.AttendanceReport, error)
	SaveBatchAttendance(ctx context.Context, req *models.BatchAttendanceRequest) error
	CreateLeaveRequest(ctx context.Context, req *models.StudentLeaveRecord) error
	ListLeaveRequests(ctx context.Context, studentID string) ([]models.StudentLeaveRecord, error)
	ReviewLeaveRequest(ctx context.Context, req *models.ReviewLeaveRequest) error
}

// ExamsRepository defines operations for Category 3.
type ExamsRepository interface {
	SetExamDates(ctx context.Context, req *models.SetExamDatesRequest) error
	GetExamDates(ctx context.Context, classID string) ([]models.ExamScheduleEntry, error)
	GetExamTimeTable(ctx context.Context, classID string) ([]models.ExamScheduleEntry, error)
	GetExamGuidelines(ctx context.Context, studentID string) (*models.ExamGuidelineData, error)
	GetResults(ctx context.Context, studentID string) ([]models.ExamScoreRecord, error)
	BatchSaveResults(ctx context.Context, req *models.BatchSaveResultsRequest) error
	GetReportCard(ctx context.Context, studentID string) (map[string]interface{}, error)
}

// FinanceRepository defines operations for Category 4.
type FinanceRepository interface {
	CreatePaymentOrder(ctx context.Context, req *models.CreatePaymentOrderRequest) (*models.PaymentOrderData, error)
	VerifyPayment(ctx context.Context, req *models.VerifyPaymentRequest) (*models.StudentPaymentTransaction, error)
	GetPaymentHistory(ctx context.Context, studentID string) ([]models.StudentPaymentTransaction, error)
	GenerateTaxReceipt(ctx context.Context, req *models.GenerateTaxReceiptRequest) (*models.TaxReceiptData, error)
}

// LogisticsRepository defines operations for Category 5.
type LogisticsRepository interface {
	GetBusTravelInfo(ctx context.Context, routeNumber string) (*models.BusTravelInfo, error)
	GetCanteenWallet(ctx context.Context, studentID string) (float64, error)
	RechargeCanteenWallet(ctx context.Context, studentID string, amount float64) (float64, error)
	GetCanteenOrders(ctx context.Context, studentID string) ([]models.CanteenMealOrder, error)
	CreateCanteenOrder(ctx context.Context, req *models.CanteenOrderRequest) (*models.CanteenMealOrder, error)
	GetHallPass(ctx context.Context, studentID string) ([]map[string]interface{}, error)
	IssueHallPass(ctx context.Context, req *models.IssueHallPassRequest, issuedBy string) (map[string]interface{}, error)
	GetHealthRecords(ctx context.Context, studentID string) (map[string]interface{}, error)
	UpdateHealthRecords(ctx context.Context, req *models.UpdateHealthRequest) error
	GetStoreProducts(ctx context.Context) ([]map[string]interface{}, error)
	GetStoreCart(ctx context.Context, studentID string) ([]models.CartItem, error)
	UpdateStoreCart(ctx context.Context, req *models.UpdateStoreCartRequest) error
	CreateStoreOrder(ctx context.Context, studentID string) (map[string]interface{}, error)
	SearchLibraryBooks(ctx context.Context, query string) ([]map[string]interface{}, error)
	GetBorrowedBooks(ctx context.Context, studentID string) ([]map[string]interface{}, error)
}

// CampusRepository defines operations for Category 6.
type CampusRepository interface {
	GetClubs(ctx context.Context) ([]map[string]interface{}, error)
	JoinClub(ctx context.Context, clubID, studentID string) error
	GetHouses(ctx context.Context) ([]map[string]interface{}, error)
	AddHousePoints(ctx context.Context, req *models.AddHousePointsRequest, teacher string) error
	GetLostItems(ctx context.Context) ([]map[string]interface{}, error)
	ReportLostItem(ctx context.Context, item map[string]interface{}) error
	ClaimLostItem(ctx context.Context, req *models.ClaimLostItemRequest) error
	GetPhotoAlbums(ctx context.Context) ([]map[string]interface{}, error)
	GetAlbumPhotos(ctx context.Context, albumID string) (map[string]interface{}, error)
}

// ParentDeskRepository defines operations for Category 7.
type ParentDeskRepository interface {
	GetParentThreads(ctx context.Context, studentID string) ([]models.ParentMessageThread, error)
	PostThreadMessage(ctx context.Context, req *models.PostParentDeskRequest, senderID, senderRole string) error
	SearchPTMTeachers(ctx context.Context) ([]models.PtmTeacherTimingData, error)
	GetPTMTeacherTiming(ctx context.Context, teacherID string) (*models.PtmTeacherTimingData, error)
	BookPTMSlot(ctx context.Context, req *models.PtmQuickBookReq) (map[string]interface{}, error)
	ReschedulePTMSlot(ctx context.Context, req *models.ReschedulePtmReq) error
	AddCalendarEvent(ctx context.Context, req *models.AddCalendarEventRequest) (string, error)
	UrgentBroadcast(ctx context.Context, req *models.UrgentNoticeBroadcastRequest) (int, error)
	PostNotice(ctx context.Context, req *models.PostNoticeRequest) (string, error)
	TrackNotification(ctx context.Context, refID string) (*models.NotificationTrackingReport, error)
}

// StaffRepository defines operations for Category 8.
type StaffRepository interface {
	GetSchoolInfo(ctx context.Context) (*models.SchoolInfoData, error)
	AddStaff(ctx context.Context, req *models.AddStaffRequest) (string, error)
	UpdateStaff(ctx context.Context, req *models.UpdateStaffRequest) error
	DeleteStaff(ctx context.Context, staffID string) error
}
