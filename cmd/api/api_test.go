package main

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/golang-jwt/jwt/v5"
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

func generateTestToken(userID, role, classID, sectionID, jwtSecret string) string {
	claims := &models.UserClaims{
		UserID:           userID,
		Role:             role,
		ClassID:          classID,
		SectionID:        sectionID,
		LinkedStudentIDs: []string{userID},
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "nexaCampus-school-portal",
			Subject:   userID,
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenStr, _ := token.SignedString([]byte(jwtSecret))
	return tokenStr
}

func computeTestPaymentHMAC(orderID, paymentID, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(orderID + "|" + paymentID))
	return hex.EncodeToString(mac.Sum(nil))
}

func setupTestRouter(t *testing.T) (*chi.Mux, *config.Config, func()) {
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

	r.Route("/api/v1", func(api chi.Router) {
		api.Use(appmiddleware.JWTAuth(cfg.JWTSecret))

		// Category 1
		api.Get("/studentData", studentHandler.GetStudentData)
		api.With(appmiddleware.RequireRoles("admin", "superadmin", "teacher")).Post("/newStudentData", studentHandler.NewStudentData)
		api.Put("/updateStudentData", studentHandler.UpdateStudentData)

		// Category 2
		api.Get("/timeTable", academicsHandler.GetTimeTable)
		api.With(appmiddleware.RequireRoles("admin", "superadmin", "teacher")).Put("/updateTimeTable", academicsHandler.UpdateTimeTable)
		api.Get("/labTimeTable", academicsHandler.GetLabTimeTable)
		api.Get("/syllabus", academicsHandler.GetSyllabus)
		api.Get("/dairy", academicsHandler.GetDiary)
		api.With(appmiddleware.RequireRoles("admin", "superadmin", "teacher")).Post("/dairy", academicsHandler.AddDiary)
		api.Get("/homework", academicsHandler.GetHomework)
		api.With(appmiddleware.RequireRoles("admin", "superadmin", "teacher")).Post("/homework", academicsHandler.PostHomework)
		api.Post("/homework/submit", academicsHandler.SubmitHomework)
		api.Get("/attendance", attendanceHandler.GetAttendance)
		api.With(appmiddleware.RequireRoles("admin", "superadmin", "teacher")).Post("/attendance", attendanceHandler.BatchAttendance)
		api.Post("/leaveRequest", attendanceHandler.CreateLeaveRequest)
		api.Get("/leaveRequest", attendanceHandler.ListLeaveRequests)
		api.With(appmiddleware.RequireRoles("admin", "superadmin", "teacher")).Put("/leaveRequest/review", attendanceHandler.ReviewLeaveRequest)

		// Category 3
		api.With(appmiddleware.RequireRoles("admin", "superadmin", "teacher")).Post("/setExamDates", academicsHandler.SetExamDates)
		api.Get("/getExamDates", academicsHandler.GetExamDates)
		api.Get("/examTimeTable", academicsHandler.GetExamTimeTable)
		api.Get("/exam", academicsHandler.GetExamGuidelines)
		api.Get("/results", academicsHandler.GetResults)
		api.With(appmiddleware.RequireRoles("admin", "superadmin", "teacher")).Post("/results", academicsHandler.BatchSaveResults)
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
		api.With(appmiddleware.RequireRoles("admin", "superadmin", "teacher")).Post("/addHousePoints", campusHandler.AddHousePoints)
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
		api.With(appmiddleware.RequireRoles("admin", "superadmin", "teacher")).Post("/noticeMessageUrgentBroadcast", ptmHandler.UrgentNoticeBroadcast)
		api.With(appmiddleware.RequireRoles("admin", "superadmin", "teacher")).Post("/postNotice", ptmHandler.PostNotice)
		api.Get("/trackNotificationsReference", ptmHandler.TrackNotification)

		// Category 8
		api.Get("/schoolInfo", auditStaffHandler.GetSchoolInfo)
		api.With(appmiddleware.RequireRoles("admin", "superadmin")).Post("/addStaff", auditStaffHandler.AddStaff)
		api.With(appmiddleware.RequireRoles("admin", "superadmin")).Put("/updateStaff", auditStaffHandler.UpdateStaff)
		api.With(appmiddleware.RequireRoles("admin", "superadmin")).Delete("/deleteStaff", auditStaffHandler.DeleteStaff)
		api.Get("/health", auditStaffHandler.GetHealth)
	})

	teardown := func() {
		_ = sqliteDB.Close()
		cacheEngine.Close()
		_ = os.Remove(tmpDB)
	}

	return r, cfg, teardown
}

func TestCategory1StudentData(t *testing.T) {
	r, cfg, teardown := setupTestRouter(t)
	defer teardown()

	token := generateTestToken("STU1001", "student", "10", "A", cfg.JWTSecret)
	req := httptest.NewRequest("GET", "/api/v1/studentData?student_id=STU1001", nil)
	req.Header.Set("Authorization", "Bearer "+token)
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
	r, cfg, teardown := setupTestRouter(t)
	defer teardown()

	token := generateTestToken("STU1001", "student", "10", "A", cfg.JWTSecret)
	req := httptest.NewRequest("GET", "/api/v1/timeTable?class_id=10&section_id=A", nil)
	req.Header.Set("Authorization", "Bearer "+token)
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
	r, cfg, teardown := setupTestRouter(t)
	defer teardown()

	token := generateTestToken("TEA1001", "teacher", "10", "A", cfg.JWTSecret)
	payload := `{"class_id":"10","section_id":"A","remark":"Excellent lab work","conduct":"Positive"}`
	req := httptest.NewRequest("POST", "/api/v1/dairy", bytes.NewBufferString(payload))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
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
	r, cfg, teardown := setupTestRouter(t)
	defer teardown()

	token := generateTestToken("STU1001", "student", "10", "A", cfg.JWTSecret)
	orderPayload := `{"student_id":"STU1001","amount":15000,"fee_type":"Tuition","description":"Term 2 Fees"}`
	req := httptest.NewRequest("POST", "/api/v1/payment", bytes.NewBufferString(orderPayload))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
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
	r, cfg, teardown := setupTestRouter(t)
	defer teardown()

	token := generateTestToken("STU1001", "student", "10", "A", cfg.JWTSecret)
	req := httptest.NewRequest("GET", "/api/v1/schoolInfo", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", w.Code)
	}

	// Test health metrics
	req = httptest.NewRequest("GET", "/api/v1/health", nil)
	req.Header.Set("Authorization", "Bearer "+token)
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

func TestSecurity_UnauthenticatedRequestRejected(t *testing.T) {
	r, _, teardown := setupTestRouter(t)
	defer teardown()

	req := httptest.NewRequest("GET", "/api/v1/studentData?student_id=STU1001", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 Unauthorized for unauthenticated request, got %d", w.Code)
	}
}

func TestSecurity_IDOR_CrossStudentAccessBlocked(t *testing.T) {
	r, cfg, teardown := setupTestRouter(t)
	defer teardown()

	// Token belongs to STU1001
	token := generateTestToken("STU1001", "student", "10", "A", cfg.JWTSecret)

	// Attempt to access STU9999
	req := httptest.NewRequest("GET", "/api/v1/studentData?student_id=STU9999", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403 Forbidden for IDOR attempt, got %d: %s", w.Code, w.Body.String())
	}
}

func TestSecurity_RBAC_RoleEnforcement(t *testing.T) {
	r, cfg, teardown := setupTestRouter(t)
	defer teardown()

	// 1. Student attempting to perform admin-only operation (add staff)
	studentToken := generateTestToken("STU1001", "student", "10", "A", cfg.JWTSecret)
	staffPayload := `{"name":"Mr. Hacker","role":"Teacher","department":"Math","email":"hack@school.edu"}`
	req := httptest.NewRequest("POST", "/api/v1/addStaff", bytes.NewBufferString(staffPayload))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+studentToken)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403 Forbidden when student calls addStaff, got %d", w.Code)
	}

	// 2. Student attempting to post teacher dairy
	dairyPayload := `{"class_id":"10","section_id":"A","remark":"Falsified remark","conduct":"Fake"}`
	req = httptest.NewRequest("POST", "/api/v1/dairy", bytes.NewBufferString(dairyPayload))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+studentToken)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403 Forbidden when student calls post dairy, got %d", w.Code)
	}

	// 3. Admin successfully adding staff
	adminToken := generateTestToken("ADM001", "admin", "", "", cfg.JWTSecret)
	req = httptest.NewRequest("POST", "/api/v1/addStaff", bytes.NewBufferString(staffPayload))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+adminToken)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated && w.Code != http.StatusOK {
		t.Fatalf("expected 201/200 for admin addStaff, got %d: %s", w.Code, w.Body.String())
	}
}

func TestSecurity_PaymentHMACVerification(t *testing.T) {
	r, cfg, teardown := setupTestRouter(t)
	defer teardown()

	studentToken := generateTestToken("STU1001", "student", "10", "A", cfg.JWTSecret)
	orderID := "order_test_123"
	paymentID := "pay_test_456"

	// 1. Invalid / forged signature rejected
	forgedPayload, _ := json.Marshal(models.VerifyPaymentRequest{
		OrderID:   orderID,
		PaymentID: paymentID,
		Signature: "forged_invalid_signature_hex",
		StudentID: "STU1001",
	})
	req := httptest.NewRequest("POST", "/api/v1/payment/verify", bytes.NewBuffer(forgedPayload))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+studentToken)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 Bad Request for forged HMAC signature, got %d: %s", w.Code, w.Body.String())
	}

	// 2. Valid signature accepted
	validSig := computeTestPaymentHMAC(orderID, paymentID, cfg.RazorpayKeySecret)
	validPayload, _ := json.Marshal(models.VerifyPaymentRequest{
		OrderID:   orderID,
		PaymentID: paymentID,
		Signature: validSig,
		StudentID: "STU1001",
	})
	req = httptest.NewRequest("POST", "/api/v1/payment/verify", bytes.NewBuffer(validPayload))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+studentToken)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for valid HMAC signature, got %d: %s", w.Code, w.Body.String())
	}
}
