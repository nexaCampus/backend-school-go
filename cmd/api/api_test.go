package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/nexaCampus/backend-school-go/config"
	"github.com/nexaCampus/backend-school-go/internal/audit"
	"github.com/nexaCampus/backend-school-go/internal/cache"
	"github.com/nexaCampus/backend-school-go/internal/database"
	"github.com/nexaCampus/backend-school-go/internal/handlers"
	appmiddleware "github.com/nexaCampus/backend-school-go/internal/middleware"
	"github.com/nexaCampus/backend-school-go/internal/models"
	"github.com/nexaCampus/backend-school-go/internal/repository/firebase"
	"github.com/nexaCampus/backend-school-go/internal/repository/sqlite"
	"github.com/nexaCampus/backend-school-go/internal/services"
)

func setupTestRouter(t *testing.T) (*chi.Mux, func()) {
	tmpDB := "test_nexacampus.db"
	_ = os.Remove(tmpDB)

	sqliteDB, err := database.InitSQLite(tmpDB)
	if err != nil {
		t.Fatalf("failed to init sqlite: %v", err)
	}

	cfg := config.Load()
	sqliteRepo := sqlite.NewRepository(sqliteDB)
	fbClient := database.NewFirebaseClient(cfg)
	firebaseRepo := firebase.NewRepository(fbClient)

	cacheEngine, err := cache.NewRistrettoCache(64)
	if err != nil {
		t.Fatalf("failed to init cache: %v", err)
	}

	auditor := audit.NewAuditor(500)

	svc := services.NewService(
		sqliteRepo,
		sqliteRepo,
		sqliteRepo,
		sqliteRepo,
		sqliteRepo,
		sqliteRepo,
		sqliteRepo,
		sqliteRepo,
		firebaseRepo,
		cacheEngine,
		auditor,
		cfg,
	)

	studentHandler := handlers.NewStudentAPIHandler(svc)
	academicsHandler := handlers.NewAcademicsAPIHandler(svc)
	attendanceHandler := handlers.NewAttendanceAPIHandler(svc)
	financeHandler := handlers.NewFinanceAPIHandler(svc)
	healthHandler := handlers.NewHealthLogisticsAPIHandler(svc)
	campusHandler := handlers.NewCampusAPIHandler(svc)
	ptmHandler := handlers.NewPtmAPIHandler(svc)
	auditStaffHandler := handlers.NewAuditStaffAPIHandler(svc, "sqlite_local")

	r := chi.NewRouter()
	r.Use(appmiddleware.SecurityHeaders)
	r.Use(appmiddleware.JWTAuth(cfg.JWTSecret))

	r.Route("/api/v1", func(api chi.Router) {
		// Category 1
		api.Get("/studentData", studentHandler.GetStudentData)
		api.Post("/newStudentData", studentHandler.NewStudentData)
		api.Put("/updateStudentData", studentHandler.UpdateStudentData)

		// Category 2
		api.Get("/timeTable", academicsHandler.GetTimeTable)
		api.Put("/updateTimeTable", academicsHandler.UpdateTimeTable)
		api.Get("/labTimeTable", academicsHandler.GetLabTimeTable)
		api.Get("/syllabus", academicsHandler.GetSyllabus)
		api.Get("/dairy", academicsHandler.GetDiary)
		api.Post("/dairy", academicsHandler.AddDiary)
		api.Get("/homework", academicsHandler.GetHomework)
		api.Post("/homework", academicsHandler.PostHomework)
		api.Post("/homework/submit", academicsHandler.SubmitHomework)
		api.Get("/attendance", attendanceHandler.GetAttendance)
		api.Post("/attendance", attendanceHandler.BatchAttendance)
		api.Post("/leaveRequest", attendanceHandler.CreateLeaveRequest)
		api.Get("/leaveRequest", attendanceHandler.ListLeaveRequests)
		api.Put("/leaveRequest/review", attendanceHandler.ReviewLeaveRequest)

		// Category 3
		api.Post("/setExamDates", academicsHandler.SetExamDates)
		api.Get("/getExamDates", academicsHandler.GetExamDates)
		api.Get("/examTimeTable", academicsHandler.GetExamTimeTable)
		api.Get("/exam", academicsHandler.GetExamGuidelines)
		api.Get("/results", academicsHandler.GetResults)
		api.Post("/results", academicsHandler.BatchSaveResults)
		api.Get("/reportCard", academicsHandler.GetReportCard)

		// Category 4
		api.Post("/payment", financeHandler.CreatePayment)
		api.Post("/payment/verify", financeHandler.VerifyPayment)
		api.Get("/paymentHistory", financeHandler.GetPaymentHistory)
		api.Post("/taxReceipt", financeHandler.GenerateTaxReceipt)

		// Category 5
		api.Get("/busTravel", healthHandler.GetBusTravel)
		api.Get("/canteenWallet", healthHandler.GetCanteenWallet)
		api.Post("/canteenWallet/recharge", healthHandler.RechargeCanteenWallet)
		api.Get("/canteenFoodOrdered", healthHandler.GetCanteenOrders)
		api.Post("/canteenFoodOrdered", healthHandler.CreateCanteenOrder)
		api.Get("/hallPass", healthHandler.GetHallPass)
		api.Post("/hallPass", healthHandler.IssueHallPass)
		api.Get("/health", healthHandler.GetHealthRecords)
		api.Get("/studentHealthInfo", healthHandler.GetHealthRecords)
		api.Post("/health", healthHandler.UpdateHealthRecords)
		api.Get("/schoolStore", healthHandler.GetStoreProducts)
		api.Get("/schoolStoreItems", healthHandler.GetStoreProducts)
		api.Get("/schoolStoreCart", healthHandler.GetStoreCart)
		api.Post("/schoolStoreCart", healthHandler.UpdateStoreCart)
		api.Post("/schoolStoreAddOrder", healthHandler.CreateStoreOrder)
		api.Post("/schoolStoreOnlinePayments", healthHandler.StoreOnlinePayments)
		api.Get("/library", healthHandler.SearchLibrary)

		// Category 6
		api.Get("/clubsAndActivities", campusHandler.GetClubs)
		api.Post("/joinClubs", campusHandler.JoinClub)
		api.Get("/houses", campusHandler.GetHouses)
		api.Get("/housePoints", campusHandler.GetHouses)
		api.Post("/addHousePoints", campusHandler.AddHousePoints)
		api.Get("/lostItems", campusHandler.GetLostItems)
		api.Post("/reportLostItems", campusHandler.ReportLostItem)
		api.Post("/claimItems", campusHandler.ClaimLostItem)
		api.Get("/photoGallery", campusHandler.GetPhotoAlbums)
		api.Get("/photoGalleryAlbums", campusHandler.GetPhotoAlbums)

		// Category 7
		api.Get("/parentDesk", ptmHandler.GetParentThreads)
		api.Get("/parentRecentThreads", ptmHandler.GetParentThreads)
		api.Post("/parentDesk", ptmHandler.PostParentMessage)
		api.Get("/ptmSearchTeachers", ptmHandler.SearchPTMTeachers)
		api.Get("/ptmTeacherTiming", ptmHandler.SearchPTMTeachers)
		api.Post("/ptmQuickBook", ptmHandler.BookPTMSlot)
		api.Put("/reschedulePtmTimings", ptmHandler.ReschedulePTMSlot)
		api.Post("/addToCalender", ptmHandler.AddToCalendar)
		api.Post("/noticeMessageUrgentBroadcast", ptmHandler.UrgentNoticeBroadcast)
		api.Post("/postNotice", ptmHandler.PostNotice)
		api.Get("/trackNotificationsReference", ptmHandler.TrackNotification)

		// Category 8
		api.Get("/schoolInfo", auditStaffHandler.GetSchoolInfo)
		api.Post("/addStaff", auditStaffHandler.AddStaff)
		api.Put("/updateStaff", auditStaffHandler.UpdateStaff)
		api.Delete("/deleteStaff", auditStaffHandler.DeleteStaff)
		api.Get("/health", auditStaffHandler.GetHealth)
	})

	teardown := func() {
		_ = sqliteDB.Close()
		cacheEngine.Close()
		_ = os.Remove(tmpDB)
	}

	return r, teardown
}

func TestCategory1StudentData(t *testing.T) {
	r, teardown := setupTestRouter(t)
	defer teardown()

	req := httptest.NewRequest("GET", "/api/v1/studentData?student_id=STU1001", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d. Body: %s", w.Code, w.Body.String())
	}

	var resp models.APIResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if !resp.Success {
		t.Fatalf("expected success true")
	}
}

func TestCategory2AcademicsAndTimeTable(t *testing.T) {
	r, teardown := setupTestRouter(t)
	defer teardown()

	req := httptest.NewRequest("GET", "/api/v1/timeTable?class_id=10&section_id=A", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", w.Code)
	}

	var resp models.APIResponse
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if !resp.Success {
		t.Fatalf("expected success true")
	}
}

func TestMutationReturnsReferenceNo(t *testing.T) {
	r, teardown := setupTestRouter(t)
	defer teardown()

	payload := `{"class_id":"10","section_id":"A","remark":"Excellent lab work","conduct":"Positive"}`
	req := httptest.NewRequest("POST", "/api/v1/dairy", bytes.NewBufferString(payload))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201 Created, got %d: %s", w.Code, w.Body.String())
	}

	var resp models.APIResponse
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if !resp.Success {
		t.Fatalf("expected success true")
	}
	if !strings.HasPrefix(resp.ReferenceNo, "NEXA-") {
		t.Fatalf("expected reference number starting with NEXA-, got %s", resp.ReferenceNo)
	}
}

func TestCategory4PaymentAndGST(t *testing.T) {
	r, teardown := setupTestRouter(t)
	defer teardown()

	orderPayload := `{"student_id":"STU1001","amount":15000,"fee_type":"Tuition","description":"Term 2 Fees"}`
	req := httptest.NewRequest("POST", "/api/v1/payment", bytes.NewBufferString(orderPayload))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d: %s", w.Code, w.Body.String())
	}

	var resp models.APIResponse
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if !resp.Success || !strings.HasPrefix(resp.ReferenceNo, "NEXA-") {
		t.Fatalf("invalid payment response: %+v", resp)
	}
}

func TestCategory8SchoolInfoAndHealthMetrics(t *testing.T) {
	r, teardown := setupTestRouter(t)
	defer teardown()

	req := httptest.NewRequest("GET", "/api/v1/schoolInfo", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", w.Code)
	}

	// Test health metrics
	req = httptest.NewRequest("GET", "/api/v1/health", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", w.Code)
	}

	var resp models.APIResponse
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if !resp.Success {
		t.Fatalf("expected health status success")
	}
}
