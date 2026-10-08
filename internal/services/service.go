package services

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/nexaCampus/backend-school-go/internal/audit"
	"github.com/nexaCampus/backend-school-go/internal/cache"
	"github.com/nexaCampus/backend-school-go/internal/config"
	"github.com/nexaCampus/backend-school-go/internal/models"
	"github.com/nexaCampus/backend-school-go/internal/repository"
	"github.com/nexaCampus/backend-school-go/internal/repository/firebase"
)

// Service encapsulates the business logic, singleflight-protected caching, and audit logging.
type Service struct {
	studentRepo    repository.StudentRepository
	academicsRepo  repository.AcademicsRepository
	examsRepo      repository.ExamsRepository
	financeRepo    repository.FinanceRepository
	logisticsRepo  repository.LogisticsRepository
	campusRepo     repository.CampusRepository
	parentDeskRepo repository.ParentDeskRepository
	staffRepo      repository.StaffRepository
	firebaseRepo   *firebase.Repository
	cache          cache.Cache
	auditor        *audit.Auditor
	cfg            *config.Config
}

// NewService constructs the high-concurrency service orchestrator.
func NewService(
	studentRepo repository.StudentRepository,
	academicsRepo repository.AcademicsRepository,
	examsRepo repository.ExamsRepository,
	financeRepo repository.FinanceRepository,
	logisticsRepo repository.LogisticsRepository,
	campusRepo repository.CampusRepository,
	parentDeskRepo repository.ParentDeskRepository,
	staffRepo repository.StaffRepository,
	firebaseRepo *firebase.Repository,
	c cache.Cache,
	auditor *audit.Auditor,
	cfg *config.Config,
) *Service {
	return &Service{
		studentRepo:    studentRepo,
		academicsRepo:  academicsRepo,
		examsRepo:      examsRepo,
		financeRepo:    financeRepo,
		logisticsRepo:  logisticsRepo,
		campusRepo:     campusRepo,
		parentDeskRepo: parentDeskRepo,
		staffRepo:      staffRepo,
		firebaseRepo:   firebaseRepo,
		cache:          c,
		auditor:        auditor,
		cfg:            cfg,
	}
}

// ==========================================
// Category 1: Student Identity & Profiles
// ==========================================

// Route 1: GET /api/v1/studentData [TTL: 5m]
func (s *Service) GetStudentData(ctx context.Context, studentID string) (*models.StudentData, error) {
	cacheKey := fmt.Sprintf("student:%s:profile", studentID)
	val, err := s.cache.GetOrCompute(cacheKey, 5*time.Minute, func() (interface{}, error) {
		return s.studentRepo.GetStudent(ctx, studentID)
	})
	if err != nil {
		return nil, err
	}
	return val.(*models.StudentData), nil
}

// Route 2: POST /api/v1/newStudentData [Mutation, Invalidate Cache]
func (s *Service) NewStudentData(ctx context.Context, req *models.NewStudentRequest, actorID, actorRole, ip string) (*models.StudentData, string, error) {
	refNo := s.auditor.RecordMutation(actorID, actorRole, "CREATE_STUDENT", "students", req, ip)
	stu, err := s.studentRepo.CreateStudent(ctx, req)
	if err != nil {
		return nil, refNo, err
	}
	s.cache.InvalidatePrefix(fmt.Sprintf("student:%s", req.StudentID))
	return stu, refNo, nil
}

// Route 3: PUT /api/v1/updateStudentData [Mutation, Invalidate Cache]
func (s *Service) UpdateStudentData(ctx context.Context, req *models.UpdateStudentRequest, actorID, actorRole, ip string) (*models.StudentData, string, error) {
	refNo := s.auditor.RecordMutation(actorID, actorRole, "UPDATE_STUDENT", "students", req, ip)
	stu, err := s.studentRepo.UpdateStudent(ctx, req)
	if err != nil {
		return nil, refNo, err
	}
	s.cache.InvalidatePrefix(fmt.Sprintf("student:%s", req.StudentID))
	return stu, refNo, nil
}

// ==========================================
// Category 2: Academics, Time Tables & Classroom Operations
// ==========================================

// Route 4: GET /api/v1/timeTable [TTL: 2h]
func (s *Service) GetTimeTable(ctx context.Context, classID, sectionID string) (*models.TimeTableData, error) {
	cacheKey := fmt.Sprintf("timetable:%s:%s", classID, sectionID)
	val, err := s.cache.GetOrCompute(cacheKey, 2*time.Hour, func() (interface{}, error) {
		return s.academicsRepo.GetTimeTable(ctx, classID, sectionID)
	})
	if err != nil {
		return nil, err
	}
	return val.(*models.TimeTableData), nil
}

// Route 5: PUT /api/v1/updateTimeTable [Mutation, Invalidate Cache]
func (s *Service) UpdateTimeTable(ctx context.Context, req *models.UpdateTimeTableRequest, actorID, actorRole, ip string) (string, error) {
	refNo := s.auditor.RecordMutation(actorID, actorRole, "UPDATE_TIMETABLE", "timetables", req, ip)
	err := s.academicsRepo.UpdateTimeTable(ctx, req)
	if err != nil {
		return refNo, err
	}
	s.cache.Delete(fmt.Sprintf("timetable:%s:%s", req.ClassID, req.SectionID))
	return refNo, nil
}

// Route 6: GET /api/v1/labTimeTable [TTL: 2h]
func (s *Service) GetLabTimeTable(ctx context.Context, classID string) ([]models.LabSchedule, error) {
	cacheKey := fmt.Sprintf("lab:schedule:%s", classID)
	val, err := s.cache.GetOrCompute(cacheKey, 2*time.Hour, func() (interface{}, error) {
		return s.academicsRepo.GetLabSchedule(ctx, classID)
	})
	if err != nil {
		return nil, err
	}
	return val.([]models.LabSchedule), nil
}

// Route 7: GET /api/v1/syllabus [TTL: 12h]
func (s *Service) GetSyllabus(ctx context.Context, classID string) ([]models.SubjectSyllabus, error) {
	cacheKey := fmt.Sprintf("syllabus:%s", classID)
	val, err := s.cache.GetOrCompute(cacheKey, 12*time.Hour, func() (interface{}, error) {
		return s.academicsRepo.GetSyllabus(ctx, classID)
	})
	if err != nil {
		return nil, err
	}
	return val.([]models.SubjectSyllabus), nil
}

// Route 8: GET /api/v1/dairy [TTL: 5m]
func (s *Service) GetDiary(ctx context.Context, classID, sectionID, studentID string) ([]models.DiaryRemark, error) {
	cacheKey := fmt.Sprintf("diary:%s:%s:%s", classID, sectionID, studentID)
	val, err := s.cache.GetOrCompute(cacheKey, 5*time.Minute, func() (interface{}, error) {
		return s.academicsRepo.GetDiaryRemarks(ctx, classID, sectionID, studentID)
	})
	if err != nil {
		return nil, err
	}
	return val.([]models.DiaryRemark), nil
}

// Route 9: POST /api/v1/dairy [Mutation, Invalidate Cache]
func (s *Service) AddDiary(ctx context.Context, req *models.PostDiaryRequest, teacherName, actorID, actorRole, ip string) (string, error) {
	refNo := s.auditor.RecordMutation(actorID, actorRole, "ADD_DIARY", "diary_remarks", req, ip)
	remark := &models.DiaryRemark{
		ID:        fmt.Sprintf("DRY-%d", time.Now().UnixNano()%100000),
		ClassID:   req.ClassID,
		SectionID: req.SectionID,
		StudentID: req.StudentID,
		Teacher:   teacherName,
		Remark:    req.Remark,
		Conduct:   req.Conduct,
		Date:      time.Now().Format("2006-01-02"),
		CreatedAt: time.Now(),
	}
	err := s.academicsRepo.AddDiaryRemark(ctx, remark)
	if err != nil {
		return refNo, err
	}
	s.cache.InvalidatePrefix("diary:")
	return refNo, nil
}

// Route 10: GET /api/v1/homework [TTL: 5m]
func (s *Service) GetHomework(ctx context.Context, classID, sectionID string) ([]models.HomeworkAssignment, error) {
	cacheKey := fmt.Sprintf("homework:%s:%s", classID, sectionID)
	val, err := s.cache.GetOrCompute(cacheKey, 5*time.Minute, func() (interface{}, error) {
		return s.academicsRepo.GetHomework(ctx, classID, sectionID)
	})
	if err != nil {
		return nil, err
	}
	return val.([]models.HomeworkAssignment), nil
}

// Route 11: POST /api/v1/homework [Mutation, Invalidate Cache]
func (s *Service) PostHomework(ctx context.Context, req *models.PostHomeworkRequest, teacherName, actorID, actorRole, ip string) (string, error) {
	refNo := s.auditor.RecordMutation(actorID, actorRole, "POST_HOMEWORK", "homework_assignments", req, ip)
	hw := &models.HomeworkAssignment{
		ID:            fmt.Sprintf("HW-%d", time.Now().UnixNano()%100000),
		ClassID:       req.ClassID,
		SectionID:     req.SectionID,
		Subject:       req.Subject,
		Title:         req.Title,
		Description:   req.Description,
		Teacher:       teacherName,
		DueDate:       req.DueDate,
		AttachmentURL: req.AttachmentURL,
		CreatedAt:     time.Now(),
	}
	err := s.academicsRepo.PostHomework(ctx, hw)
	if err != nil {
		return refNo, err
	}
	s.cache.Delete(fmt.Sprintf("homework:%s:%s", req.ClassID, req.SectionID))
	return refNo, nil
}

// Route 12: POST /api/v1/homework/submit [No Cache, Mutation]
func (s *Service) SubmitHomework(ctx context.Context, hwID, studentID, fileURL, ip string) (string, error) {
	sub := &models.HomeworkSubmissionEntry{
		HomeworkID:  hwID,
		StudentID:   studentID,
		FileURL:     fileURL,
		SubmittedAt: time.Now(),
	}
	refNo := s.auditor.RecordMutation(studentID, "student", "SUBMIT_HOMEWORK", "homework_submissions", sub, ip)
	err := s.academicsRepo.SubmitHomework(ctx, sub)
	return refNo, err
}

// Route 13: GET /api/v1/attendance [TTL: 60s]
func (s *Service) GetAttendance(ctx context.Context, studentID string, month, year int) (*models.AttendanceReport, error) {
	cacheKey := fmt.Sprintf("attendance:%s:%d:%d", studentID, year, month)
	val, err := s.cache.GetOrCompute(cacheKey, 60*time.Second, func() (interface{}, error) {
		return s.academicsRepo.GetAttendance(ctx, studentID, month, year)
	})
	if err != nil {
		return nil, err
	}
	return val.(*models.AttendanceReport), nil
}

// Route 14: POST /api/v1/attendance [Mutation, Invalidate Cache]
func (s *Service) BatchAttendance(ctx context.Context, req *models.BatchAttendanceRequest, actorID, actorRole, ip string) (string, error) {
	refNo := s.auditor.RecordMutation(actorID, actorRole, "BATCH_ATTENDANCE", "attendance_records", req, ip)
	err := s.academicsRepo.SaveBatchAttendance(ctx, req)
	if err != nil {
		return refNo, err
	}
	s.cache.InvalidatePrefix("attendance:")
	return refNo, nil
}

// Route 15: POST /api/v1/leaveRequest [No Cache, Mutation]
func (s *Service) CreateLeaveRequest(ctx context.Context, req *models.StudentLeaveRecord, actorID, actorRole, ip string) (string, error) {
	req.ID = fmt.Sprintf("LEV-%d", time.Now().UnixNano()%100000)
	req.CreatedAt = time.Now()
	refNo := s.auditor.RecordMutation(actorID, actorRole, "CREATE_LEAVE", "leave_requests", req, ip)
	err := s.academicsRepo.CreateLeaveRequest(ctx, req)
	if err != nil {
		return refNo, err
	}
	s.cache.InvalidatePrefix(fmt.Sprintf("leave:%s", req.StudentID))
	return refNo, nil
}

// Route 16: GET /api/v1/leaveRequest [TTL: 1m]
func (s *Service) ListLeaveRequests(ctx context.Context, studentID string) ([]models.StudentLeaveRecord, error) {
	cacheKey := fmt.Sprintf("leave:%s", studentID)
	val, err := s.cache.GetOrCompute(cacheKey, 1*time.Minute, func() (interface{}, error) {
		return s.academicsRepo.ListLeaveRequests(ctx, studentID)
	})
	if err != nil {
		return nil, err
	}
	return val.([]models.StudentLeaveRecord), nil
}

// Route 17: PUT /api/v1/leaveRequest/review [Mutation, Invalidate Cache]
func (s *Service) ReviewLeaveRequest(ctx context.Context, req *models.ReviewLeaveRequest, actorID, actorRole, ip string) (string, error) {
	refNo := s.auditor.RecordMutation(actorID, actorRole, "REVIEW_LEAVE", "leave_requests", req, ip)
	err := s.academicsRepo.ReviewLeaveRequest(ctx, req)
	if err != nil {
		return refNo, err
	}
	s.cache.InvalidatePrefix("leave:")
	return refNo, nil
}

// ==========================================
// Category 3: Examinations, Report Cards & Results
// ==========================================

// Route 18: POST /api/v1/setExamDates [Mutation, Invalidate Cache]
func (s *Service) SetExamDates(ctx context.Context, req *models.SetExamDatesRequest, actorID, actorRole, ip string) (string, error) {
	refNo := s.auditor.RecordMutation(actorID, actorRole, "SET_EXAM_DATES", "exam_schedules", req, ip)
	err := s.examsRepo.SetExamDates(ctx, req)
	if err != nil {
		return refNo, err
	}
	s.cache.InvalidatePrefix(fmt.Sprintf("exam:dates:%s", req.ClassID))
	s.cache.InvalidatePrefix(fmt.Sprintf("exam:timetable:%s", req.ClassID))
	return refNo, nil
}

// Route 19: GET /api/v1/getExamDates [TTL: 2h]
func (s *Service) GetExamDates(ctx context.Context, classID string) ([]models.ExamScheduleEntry, error) {
	cacheKey := fmt.Sprintf("exam:dates:%s", classID)
	val, err := s.cache.GetOrCompute(cacheKey, 2*time.Hour, func() (interface{}, error) {
		return s.examsRepo.GetExamDates(ctx, classID)
	})
	if err != nil {
		return nil, err
	}
	return val.([]models.ExamScheduleEntry), nil
}

// Route 20: GET /api/v1/examTimeTable [TTL: 2h]
func (s *Service) GetExamTimeTable(ctx context.Context, classID string) ([]models.ExamScheduleEntry, error) {
	cacheKey := fmt.Sprintf("exam:timetable:%s", classID)
	val, err := s.cache.GetOrCompute(cacheKey, 2*time.Hour, func() (interface{}, error) {
		return s.examsRepo.GetExamTimeTable(ctx, classID)
	})
	if err != nil {
		return nil, err
	}
	return val.([]models.ExamScheduleEntry), nil
}

// Route 21: GET /api/v1/exam [TTL: 1h]
func (s *Service) GetExamGuidelines(ctx context.Context, studentID string) (*models.ExamGuidelineData, error) {
	cacheKey := fmt.Sprintf("exam:guidelines:%s", studentID)
	val, err := s.cache.GetOrCompute(cacheKey, 1*time.Hour, func() (interface{}, error) {
		return s.examsRepo.GetExamGuidelines(ctx, studentID)
	})
	if err != nil {
		return nil, err
	}
	return val.(*models.ExamGuidelineData), nil
}

// Route 22: GET /api/v1/results [TTL: 10m]
func (s *Service) GetResults(ctx context.Context, studentID string) ([]models.ExamScoreRecord, error) {
	cacheKey := fmt.Sprintf("exam:results:%s", studentID)
	val, err := s.cache.GetOrCompute(cacheKey, 10*time.Minute, func() (interface{}, error) {
		return s.examsRepo.GetResults(ctx, studentID)
	})
	if err != nil {
		return nil, err
	}
	return val.([]models.ExamScoreRecord), nil
}

// Route 23: POST /api/v1/results [Mutation, Invalidate Cache]
func (s *Service) BatchSaveResults(ctx context.Context, req *models.BatchSaveResultsRequest, actorID, actorRole, ip string) (string, error) {
	refNo := s.auditor.RecordMutation(actorID, actorRole, "BATCH_RESULTS", "exam_results", req, ip)
	err := s.examsRepo.BatchSaveResults(ctx, req)
	if err != nil {
		return refNo, err
	}
	s.cache.InvalidatePrefix("exam:results:")
	s.cache.InvalidatePrefix("reportcard:")
	return refNo, nil
}

// Route 24: GET /api/v1/reportCard [TTL: 15m]
func (s *Service) GetReportCard(ctx context.Context, studentID string) (map[string]interface{}, error) {
	cacheKey := fmt.Sprintf("reportcard:%s", studentID)
	val, err := s.cache.GetOrCompute(cacheKey, 15*time.Minute, func() (interface{}, error) {
		return s.examsRepo.GetReportCard(ctx, studentID)
	})
	if err != nil {
		return nil, err
	}
	return val.(map[string]interface{}), nil
}

// ==========================================
// Category 4: Payments, Invoicing & GST Compliance
// ==========================================

// Route 25: POST /api/v1/payment [No Cache, Mutation]
func (s *Service) CreatePaymentOrder(ctx context.Context, req *models.CreatePaymentOrderRequest, actorID, actorRole, ip string) (*models.PaymentOrderData, string, error) {
	refNo := s.auditor.RecordMutation(actorID, actorRole, "CREATE_PAYMENT", "payments", req, ip)
	order, err := s.financeRepo.CreatePaymentOrder(ctx, req)
	return order, refNo, err
}

// Route 26: POST /api/v1/payment/verify [No Cache, Mutation]
func (s *Service) VerifyPayment(ctx context.Context, req *models.VerifyPaymentRequest, actorID, actorRole, ip string) (*models.StudentPaymentTransaction, string, error) {
	if req.OrderID == "" || req.PaymentID == "" || req.Signature == "" {
		return nil, "", errors.New("missing required payment verification parameters: order_id, payment_id, and signature are required")
	}

	// Cryptographic HMAC-SHA256 signature verification against configured Razorpay secret
	expectedMsg := req.OrderID + "|" + req.PaymentID
	mac := hmac.New(sha256.New, []byte(s.cfg.RazorpayKeySecret))
	mac.Write([]byte(expectedMsg))
	expectedSignature := hex.EncodeToString(mac.Sum(nil))

	if !hmac.Equal([]byte(strings.ToLower(req.Signature)), []byte(strings.ToLower(expectedSignature))) {
		return nil, "", errors.New("invalid payment signature: cryptographic verification failed")
	}

	refNo := s.auditor.RecordMutation(actorID, actorRole, "VERIFY_PAYMENT", "payments", req, ip)
	txn, err := s.financeRepo.VerifyPayment(ctx, req)
	if err != nil {
		return nil, refNo, err
	}
	s.cache.Delete(fmt.Sprintf("payment:history:%s", req.StudentID))
	return txn, refNo, nil
}

// Route 27: GET /api/v1/paymentHistory [TTL: 2m]
func (s *Service) GetPaymentHistory(ctx context.Context, studentID string) ([]models.StudentPaymentTransaction, error) {
	cacheKey := fmt.Sprintf("payment:history:%s", studentID)
	val, err := s.cache.GetOrCompute(cacheKey, 2*time.Minute, func() (interface{}, error) {
		return s.financeRepo.GetPaymentHistory(ctx, studentID)
	})
	if err != nil {
		return nil, err
	}
	return val.([]models.StudentPaymentTransaction), nil
}

// Route 28: POST /api/v1/taxReceipt [No Cache, Mutation]
func (s *Service) GenerateTaxReceipt(ctx context.Context, req *models.GenerateTaxReceiptRequest, actorID, actorRole, ip string) (*models.TaxReceiptData, string, error) {
	refNo := s.auditor.RecordMutation(actorID, actorRole, "TAX_RECEIPT", "tax_receipts", req, ip)
	data, err := s.financeRepo.GenerateTaxReceipt(ctx, req)
	return data, refNo, err
}

// ==========================================
// Category 5: Campus Logistics, Canteen, Store & Health
// ==========================================

// Route 29: GET /api/v1/busTravel [TTL: 15s]
func (s *Service) GetBusTravel(ctx context.Context, routeNumber string) (*models.BusTravelInfo, error) {
	cacheKey := fmt.Sprintf("bus:travel:%s", routeNumber)
	val, err := s.cache.GetOrCompute(cacheKey, 15*time.Second, func() (interface{}, error) {
		info, err := s.logisticsRepo.GetBusTravelInfo(ctx, routeNumber)
		if err != nil {
			return nil, err
		}
		// Stream live coordinates from Firebase
		if s.firebaseRepo != nil {
			lat, lng := s.firebaseRepo.GetLiveBusCoordinates(ctx, routeNumber)
			info.CurrentLat = lat
			info.CurrentLng = lng
		}
		return info, nil
	})
	if err != nil {
		return nil, err
	}
	return val.(*models.BusTravelInfo), nil
}

// Route 30: GET /api/v1/canteenWallet [TTL: 30s]
func (s *Service) GetCanteenWallet(ctx context.Context, studentID string) (float64, error) {
	cacheKey := fmt.Sprintf("canteen:wallet:%s", studentID)
	val, err := s.cache.GetOrCompute(cacheKey, 30*time.Second, func() (interface{}, error) {
		return s.logisticsRepo.GetCanteenWallet(ctx, studentID)
	})
	if err != nil {
		return 0, err
	}
	return val.(float64), nil
}

// Route 31: POST /api/v1/canteenWallet/recharge [No Cache, Mutation]
func (s *Service) RechargeCanteenWallet(ctx context.Context, studentID string, amount float64, actorID, actorRole, ip string) (float64, string, error) {
	refNo := s.auditor.RecordMutation(actorID, actorRole, "RECHARGE_WALLET", "canteen_wallets", map[string]interface{}{"amount": amount}, ip)
	newBal, err := s.logisticsRepo.RechargeCanteenWallet(ctx, studentID, amount)
	if err != nil {
		return 0, refNo, err
	}
	s.cache.Delete(fmt.Sprintf("canteen:wallet:%s", studentID))
	return newBal, refNo, nil
}

// Route 32: GET /api/v1/canteenFoodOrdered [TTL: 30s]
func (s *Service) GetCanteenOrders(ctx context.Context, studentID string) ([]models.CanteenMealOrder, error) {
	cacheKey := fmt.Sprintf("canteen:orders:%s", studentID)
	val, err := s.cache.GetOrCompute(cacheKey, 30*time.Second, func() (interface{}, error) {
		return s.logisticsRepo.GetCanteenOrders(ctx, studentID)
	})
	if err != nil {
		return nil, err
	}
	return val.([]models.CanteenMealOrder), nil
}

// Route 33: POST /api/v1/canteenFoodOrdered [Mutation, Invalidate Cache]
func (s *Service) CreateCanteenOrder(ctx context.Context, req *models.CanteenOrderRequest, actorID, actorRole, ip string) (*models.CanteenMealOrder, string, error) {
	refNo := s.auditor.RecordMutation(actorID, actorRole, "ORDER_CANTEEN", "canteen_orders", req, ip)
	order, err := s.logisticsRepo.CreateCanteenOrder(ctx, req)
	if err != nil {
		return nil, refNo, err
	}
	s.cache.Delete(fmt.Sprintf("canteen:wallet:%s", req.StudentID))
	s.cache.Delete(fmt.Sprintf("canteen:orders:%s", req.StudentID))
	return order, refNo, nil
}

// Route 34: GET /api/v1/hallPass [TTL: 15s]
func (s *Service) GetHallPass(ctx context.Context, studentID string) ([]map[string]interface{}, error) {
	cacheKey := fmt.Sprintf("hallpass:%s", studentID)
	val, err := s.cache.GetOrCompute(cacheKey, 15*time.Second, func() (interface{}, error) {
		return s.logisticsRepo.GetHallPass(ctx, studentID)
	})
	if err != nil {
		return nil, err
	}
	return val.([]map[string]interface{}), nil
}

// Route 35: POST /api/v1/hallPass [Mutation, Invalidate Cache]
func (s *Service) IssueHallPass(ctx context.Context, req *models.IssueHallPassRequest, issuedBy, actorID, actorRole, ip string) (map[string]interface{}, string, error) {
	refNo := s.auditor.RecordMutation(actorID, actorRole, "ISSUE_HALLPASS", "hall_passes", req, ip)
	pass, err := s.logisticsRepo.IssueHallPass(ctx, req, issuedBy)
	if err != nil {
		return nil, refNo, err
	}
	s.cache.Delete(fmt.Sprintf("hallpass:%s", req.StudentID))
	return pass, refNo, nil
}

// Route 36: GET /api/v1/health & /api/v1/studentHealthInfo [TTL: 10m]
func (s *Service) GetHealthRecords(ctx context.Context, studentID string) (map[string]interface{}, error) {
	cacheKey := fmt.Sprintf("health:%s", studentID)
	val, err := s.cache.GetOrCompute(cacheKey, 10*time.Minute, func() (interface{}, error) {
		return s.logisticsRepo.GetHealthRecords(ctx, studentID)
	})
	if err != nil {
		return nil, err
	}
	return val.(map[string]interface{}), nil
}

// Route 37: POST /api/v1/health [Mutation, Invalidate Cache]
func (s *Service) UpdateHealthRecords(ctx context.Context, req *models.UpdateHealthRequest, actorID, actorRole, ip string) (string, error) {
	refNo := s.auditor.RecordMutation(actorID, actorRole, "UPDATE_HEALTH", "health_records", req, ip)
	err := s.logisticsRepo.UpdateHealthRecords(ctx, req)
	if err != nil {
		return refNo, err
	}
	s.cache.Delete(fmt.Sprintf("health:%s", req.StudentID))
	return refNo, nil
}

// Route 38: GET /api/v1/schoolStore & /api/v1/schoolStoreItems [TTL: 30m]
func (s *Service) GetStoreProducts(ctx context.Context) ([]map[string]interface{}, error) {
	cacheKey := "store:items"
	val, err := s.cache.GetOrCompute(cacheKey, 30*time.Minute, func() (interface{}, error) {
		return s.logisticsRepo.GetStoreProducts(ctx)
	})
	if err != nil {
		return nil, err
	}
	return val.([]map[string]interface{}), nil
}

// Route 39: GET & POST /api/v1/schoolStoreCart
func (s *Service) GetStoreCart(ctx context.Context, studentID string) ([]models.CartItem, error) {
	cacheKey := fmt.Sprintf("store:cart:%s", studentID)
	val, err := s.cache.GetOrCompute(cacheKey, 1*time.Minute, func() (interface{}, error) {
		return s.logisticsRepo.GetStoreCart(ctx, studentID)
	})
	if err != nil {
		return nil, err
	}
	return val.([]models.CartItem), nil
}

func (s *Service) UpdateStoreCart(ctx context.Context, req *models.UpdateStoreCartRequest, actorID, actorRole, ip string) (string, error) {
	refNo := s.auditor.RecordMutation(actorID, actorRole, "UPDATE_CART", "store_carts", req, ip)
	err := s.logisticsRepo.UpdateStoreCart(ctx, req)
	if err != nil {
		return refNo, err
	}
	s.cache.Delete(fmt.Sprintf("store:cart:%s", req.StudentID))
	return refNo, nil
}

// Route 40: POST /api/v1/schoolStoreAddOrder [Mutation, Invalidate Cache]
func (s *Service) CreateStoreOrder(ctx context.Context, studentID, actorID, actorRole, ip string) (map[string]interface{}, string, error) {
	refNo := s.auditor.RecordMutation(actorID, actorRole, "ADD_STORE_ORDER", "store_orders", map[string]string{"student_id": studentID}, ip)
	order, err := s.logisticsRepo.CreateStoreOrder(ctx, studentID)
	if err != nil {
		return nil, refNo, err
	}
	s.cache.Delete(fmt.Sprintf("store:cart:%s", studentID))
	return order, refNo, nil
}

// Route 41: POST /api/v1/schoolStoreOnlinePayments [No Cache, Mutation]
func (s *Service) StoreOnlinePayments(ctx context.Context, studentID string, amount float64, actorID, actorRole, ip string) (*models.PaymentOrderData, string, error) {
	req := &models.CreatePaymentOrderRequest{
		StudentID:   studentID,
		Amount:      amount,
		FeeType:     "Store Purchase",
		Description: "School Store Order Checkout",
	}
	return s.CreatePaymentOrder(ctx, req, actorID, actorRole, ip)
}

// Route 42: GET /api/v1/library [TTL: 15m]
func (s *Service) SearchLibrary(ctx context.Context, query, studentID string) (map[string]interface{}, error) {
	cacheKey := fmt.Sprintf("library:books:%s:%s", query, studentID)
	val, err := s.cache.GetOrCompute(cacheKey, 15*time.Minute, func() (interface{}, error) {
		books, err := s.logisticsRepo.SearchLibraryBooks(ctx, query)
		if err != nil {
			return nil, err
		}
		borrowed, _ := s.logisticsRepo.GetBorrowedBooks(ctx, studentID)
		return map[string]interface{}{
			"books":    books,
			"borrowed": borrowed,
		}, nil
	})
	if err != nil {
		return nil, err
	}
	return val.(map[string]interface{}), nil
}

// ==========================================
// Category 6: Campus Life, Houses, Lost Property & Media
// ==========================================

// Route 43: GET /api/v1/clubsAndActivities [TTL: 1h]
func (s *Service) GetClubs(ctx context.Context) ([]map[string]interface{}, error) {
	cacheKey := "clubs:list"
	val, err := s.cache.GetOrCompute(cacheKey, 1*time.Hour, func() (interface{}, error) {
		return s.campusRepo.GetClubs(ctx)
	})
	if err != nil {
		return nil, err
	}
	return val.([]map[string]interface{}), nil
}

// Route 44: POST /api/v1/joinClubs [No Cache, Mutation]
func (s *Service) JoinClub(ctx context.Context, clubID, studentID, actorID, actorRole, ip string) (string, error) {
	refNo := s.auditor.RecordMutation(actorID, actorRole, "JOIN_CLUB", "club_memberships", map[string]string{"club_id": clubID, "student_id": studentID}, ip)
	err := s.campusRepo.JoinClub(ctx, clubID, studentID)
	return refNo, err
}

// Route 45: GET /api/v1/houses & /api/v1/housePoints [TTL: 5m]
func (s *Service) GetHouses(ctx context.Context) ([]map[string]interface{}, error) {
	cacheKey := "houses:points"
	val, err := s.cache.GetOrCompute(cacheKey, 5*time.Minute, func() (interface{}, error) {
		return s.campusRepo.GetHouses(ctx)
	})
	if err != nil {
		return nil, err
	}
	return val.([]map[string]interface{}), nil
}

// Route 46: POST /api/v1/addHousePoints [Mutation, Invalidate Cache]
func (s *Service) AddHousePoints(ctx context.Context, req *models.AddHousePointsRequest, teacher, actorID, actorRole, ip string) (string, error) {
	refNo := s.auditor.RecordMutation(actorID, actorRole, "ADD_HOUSE_POINTS", "houses", req, ip)
	err := s.campusRepo.AddHousePoints(ctx, req, teacher)
	if err != nil {
		return refNo, err
	}
	s.cache.Delete("houses:points")
	return refNo, nil
}

// Route 47: GET /api/v1/lostItems [TTL: 5m]
func (s *Service) GetLostItems(ctx context.Context) ([]map[string]interface{}, error) {
	cacheKey := "lostitems:list"
	val, err := s.cache.GetOrCompute(cacheKey, 5*time.Minute, func() (interface{}, error) {
		return s.campusRepo.GetLostItems(ctx)
	})
	if err != nil {
		return nil, err
	}
	return val.([]map[string]interface{}), nil
}

// Route 48: POST /api/v1/reportLostItems [Mutation, Invalidate Cache]
func (s *Service) ReportLostItem(ctx context.Context, item map[string]interface{}, actorID, actorRole, ip string) (string, error) {
	refNo := s.auditor.RecordMutation(actorID, actorRole, "REPORT_LOST_ITEM", "lost_items", item, ip)
	err := s.campusRepo.ReportLostItem(ctx, item)
	if err != nil {
		return refNo, err
	}
	s.cache.Delete("lostitems:list")
	return refNo, nil
}

// Route 49: POST /api/v1/claimItems [No Cache, Mutation]
func (s *Service) ClaimLostItem(ctx context.Context, req *models.ClaimLostItemRequest, actorID, actorRole, ip string) (string, error) {
	refNo := s.auditor.RecordMutation(actorID, actorRole, "CLAIM_LOST_ITEM", "lost_items", req, ip)
	err := s.campusRepo.ClaimLostItem(ctx, req)
	if err != nil {
		return refNo, err
	}
	s.cache.Delete("lostitems:list")
	return refNo, nil
}

// Route 50: GET /api/v1/photoGallery & /api/v1/photoGalleryAlbums [TTL: 1h]
func (s *Service) GetPhotoAlbums(ctx context.Context) ([]map[string]interface{}, error) {
	cacheKey := "gallery:albums"
	val, err := s.cache.GetOrCompute(cacheKey, 1*time.Hour, func() (interface{}, error) {
		return s.campusRepo.GetPhotoAlbums(ctx)
	})
	if err != nil {
		return nil, err
	}
	return val.([]map[string]interface{}), nil
}

// ==========================================
// Category 7: Parent Desk, PTM & Notifications
// ==========================================

// Route 51: GET /api/v1/parentDesk & /api/v1/parentRecentThreads [TTL: 15s]
func (s *Service) GetParentThreads(ctx context.Context, studentID string) ([]models.ParentMessageThread, error) {
	cacheKey := fmt.Sprintf("parentdesk:threads:%s", studentID)
	val, err := s.cache.GetOrCompute(cacheKey, 15*time.Second, func() (interface{}, error) {
		return s.parentDeskRepo.GetParentThreads(ctx, studentID)
	})
	if err != nil {
		return nil, err
	}
	return val.([]models.ParentMessageThread), nil
}

// Route 52: POST /api/v1/parentDesk [Mutation, Invalidate Cache]
func (s *Service) PostParentMessage(ctx context.Context, req *models.PostParentDeskRequest, senderID, senderRole, ip string) (string, error) {
	refNo := s.auditor.RecordMutation(senderID, senderRole, "PARENT_DESK_MSG", "parent_threads", req, ip)
	err := s.parentDeskRepo.PostThreadMessage(ctx, req, senderID, senderRole)
	if err != nil {
		return refNo, err
	}
	s.cache.Delete(fmt.Sprintf("parentdesk:threads:%s", req.StudentID))
	return refNo, nil
}

// Route 53: GET /api/v1/ptmSearchTeachers & /api/v1/ptmTeacherTiming [TTL: 2m]
func (s *Service) SearchPTMTeachers(ctx context.Context) ([]models.PtmTeacherTimingData, error) {
	cacheKey := "ptm:teachers"
	val, err := s.cache.GetOrCompute(cacheKey, 2*time.Minute, func() (interface{}, error) {
		return s.parentDeskRepo.SearchPTMTeachers(ctx)
	})
	if err != nil {
		return nil, err
	}
	return val.([]models.PtmTeacherTimingData), nil
}

func (s *Service) GetPTMTeacherTiming(ctx context.Context, teacherID string) (*models.PtmTeacherTimingData, error) {
	cacheKey := fmt.Sprintf("ptm:timing:%s", teacherID)
	val, err := s.cache.GetOrCompute(cacheKey, 2*time.Minute, func() (interface{}, error) {
		return s.parentDeskRepo.GetPTMTeacherTiming(ctx, teacherID)
	})
	if err != nil {
		return nil, err
	}
	return val.(*models.PtmTeacherTimingData), nil
}

// Route 54: POST /api/v1/ptmQuickBook [Mutation, Invalidate Cache]
func (s *Service) BookPTMSlot(ctx context.Context, req *models.PtmQuickBookReq, actorID, actorRole, ip string) (map[string]interface{}, string, error) {
	refNo := s.auditor.RecordMutation(actorID, actorRole, "BOOK_PTM", "ptm_bookings", req, ip)
	res, err := s.parentDeskRepo.BookPTMSlot(ctx, req)
	if err != nil {
		return nil, refNo, err
	}
	s.cache.Delete("ptm:teachers")
	s.cache.Delete(fmt.Sprintf("ptm:timing:%s", req.TeacherID))
	return res, refNo, nil
}

// Route 55: PUT /api/v1/reschedulePtmTimings [Mutation, Invalidate Cache]
func (s *Service) ReschedulePTMSlot(ctx context.Context, req *models.ReschedulePtmReq, actorID, actorRole, ip string) (string, error) {
	refNo := s.auditor.RecordMutation(actorID, actorRole, "RESCHEDULE_PTM", "ptm_bookings", req, ip)
	err := s.parentDeskRepo.ReschedulePTMSlot(ctx, req)
	if err != nil {
		return refNo, err
	}
	s.cache.InvalidatePrefix("ptm:")
	return refNo, nil
}

// Route 56: POST /api/v1/addToCalender [TTL: 1h, Generates payload]
func (s *Service) AddToCalendar(ctx context.Context, req *models.AddCalendarEventRequest, actorID, actorRole, ip string) (string, string, error) {
	refNo := s.auditor.RecordMutation(actorID, actorRole, "ADD_CALENDAR", "calendar_events", req, ip)
	eventID, err := s.parentDeskRepo.AddCalendarEvent(ctx, req)
	return eventID, refNo, err
}

// Route 57: POST /api/v1/noticeMessageUrgentBroadcast [No Cache, Admin Mutation]
func (s *Service) UrgentNoticeBroadcast(ctx context.Context, req *models.UrgentNoticeBroadcastRequest, actorID, actorRole, ip string) (int, string, error) {
	refNo := s.auditor.RecordMutation(actorID, actorRole, "URGENT_BROADCAST", "notices", req, ip)
	sentCount, err := s.parentDeskRepo.UrgentBroadcast(ctx, req)
	if err != nil {
		return 0, refNo, err
	}
	// Broadcast through FCM
	if s.firebaseRepo != nil && req.SendFCM {
		_, _ = s.firebaseRepo.SendUrgentAlert(ctx, req.Title, req.Message, req.Priority, req.TargetClasses)
	}
	return sentCount, refNo, nil
}

// Route 58: POST /api/v1/postNotice [Mutation, Invalidate Cache]
func (s *Service) PostNotice(ctx context.Context, req *models.PostNoticeRequest, actorID, actorRole, ip string) (string, string, error) {
	refNo := s.auditor.RecordMutation(actorID, actorRole, "POST_NOTICE", "notices", req, ip)
	id, err := s.parentDeskRepo.PostNotice(ctx, req)
	if err != nil {
		return "", refNo, err
	}
	s.cache.InvalidatePrefix("notices:")
	return id, refNo, nil
}

// Route 59: GET /api/v1/trackNotificationsReference [TTL: 1m]
func (s *Service) TrackNotification(ctx context.Context, refID string) (*models.NotificationTrackingReport, error) {
	cacheKey := fmt.Sprintf("notif:track:%s", refID)
	val, err := s.cache.GetOrCompute(cacheKey, 1*time.Minute, func() (interface{}, error) {
		return s.parentDeskRepo.TrackNotification(ctx, refID)
	})
	if err != nil {
		return nil, err
	}
	return val.(*models.NotificationTrackingReport), nil
}

// ==========================================
// Category 8: Staff Administration & Core System
// ==========================================

// Route 60: GET /api/v1/schoolInfo [TTL: 24h]
func (s *Service) GetSchoolInfo(ctx context.Context) (*models.SchoolInfoData, error) {
	cacheKey := "school:info"
	val, err := s.cache.GetOrCompute(cacheKey, 24*time.Hour, func() (interface{}, error) {
		return s.staffRepo.GetSchoolInfo(ctx)
	})
	if err != nil {
		return nil, err
	}
	return val.(*models.SchoolInfoData), nil
}

// Route 61: POST /api/v1/addStaff [No Cache, SuperAdmin Mutation]
func (s *Service) AddStaff(ctx context.Context, req *models.AddStaffRequest, actorID, actorRole, ip string) (string, string, error) {
	refNo := s.auditor.RecordMutation(actorID, actorRole, "ADD_STAFF", "staff", req, ip)
	id, err := s.staffRepo.AddStaff(ctx, req)
	if err != nil {
		return "", refNo, err
	}
	s.cache.InvalidatePrefix("staff:")
	return id, refNo, nil
}

// Route 62: PUT /api/v1/updateStaff [Mutation, Invalidate Cache]
func (s *Service) UpdateStaff(ctx context.Context, req *models.UpdateStaffRequest, actorID, actorRole, ip string) (string, error) {
	refNo := s.auditor.RecordMutation(actorID, actorRole, "UPDATE_STAFF", "staff", req, ip)
	err := s.staffRepo.UpdateStaff(ctx, req)
	if err != nil {
		return refNo, err
	}
	s.cache.InvalidatePrefix("staff:")
	return refNo, nil
}

// Route 63: DELETE /api/v1/deleteStaff [Mutation, Invalidate Cache]
func (s *Service) DeleteStaff(ctx context.Context, staffID, actorID, actorRole, ip string) (string, error) {
	refNo := s.auditor.RecordMutation(actorID, actorRole, "DELETE_STAFF", "staff", map[string]string{"staff_id": staffID}, ip)
	err := s.staffRepo.DeleteStaff(ctx, staffID)
	if err != nil {
		return refNo, err
	}
	s.cache.InvalidatePrefix("staff:")
	return refNo, nil
}

// Route 64: GET /api/v1/health [No Cache, Live Metrics]
func (s *Service) GetHealthMetrics(uptimeSeconds float64, storageMode string) *models.HealthCheckMetrics {
	dbPing := "OK"
	cacheItems := 0
	if s.cache != nil {
		cacheItems = s.cache.Len()
	}

	return &models.HealthCheckMetrics{
		Status:           "HEALTHY",
		UptimeSeconds:    uptimeSeconds,
		GCMemoryMB:       0,
		AllocatedMB:      0,
		TotalAllocMB:     0,
		ActiveGoroutines: 0,
		StorageMode:      storageMode,
		DatabasePing:     dbPing,
		CacheItems:       cacheItems,
	}
}
