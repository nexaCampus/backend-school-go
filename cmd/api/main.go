package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"runtime/debug"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/nexaCampus/backend-school-go/config"
	"github.com/nexaCampus/backend-school-go/internal/audit"
	"github.com/nexaCampus/backend-school-go/internal/cache"
	"github.com/nexaCampus/backend-school-go/internal/database"
	"github.com/nexaCampus/backend-school-go/internal/handlers"
	appmiddleware "github.com/nexaCampus/backend-school-go/internal/middleware"
	"github.com/nexaCampus/backend-school-go/internal/repository"
	"github.com/nexaCampus/backend-school-go/internal/repository/firebase"
	"github.com/nexaCampus/backend-school-go/internal/repository/sqlite"
	"github.com/nexaCampus/backend-school-go/internal/repository/supabase"
	"github.com/nexaCampus/backend-school-go/internal/services"
)

func main() {
	log.Println("[INFO] Bootstrapping NexaCampus Ultra-High-Concurrency Backend Engine...")

	// 1. Runtime Profile & Memory Bounds Tuning
	cfg := config.Load()
	debug.SetGCPercent(cfg.GOGC)
	debug.SetMemoryLimit(cfg.MemoryLimitMB * 1024 * 1024)
	log.Printf("[RUNTIME] GOGC=%d, Hard Memory Limit=%d MB", cfg.GOGC, cfg.MemoryLimitMB)

	// 2. Dual-Storage & Fallback Initialization
	sqliteDB, err := database.InitSQLite(cfg.SQLitePath)
	if err != nil {
		log.Fatalf("[FATAL] Failed to initialize SQLite storage: %v", err)
	}
	defer sqliteDB.Close()

	sqliteRepo := sqlite.NewRepository(sqliteDB)
	var activeStudentRepo repository.StudentRepository = sqliteRepo

	if cfg.StorageMode == "hybrid" && cfg.DatabaseURL != "" {
		pgPool, pgErr := database.InitSupabase(cfg)
		if pgErr == nil {
			log.Println("[INFO] Supabase PostgreSQL connected. Operating in hybrid dual-storage mode.")
			supaRepo := supabase.NewRepository(pgPool, sqliteRepo)
			activeStudentRepo = supaRepo
		} else {
			log.Printf("[WARN] Supabase unreachable (%v). Seamlessly running in local SQLite WAL mode.", pgErr)
		}
	}

	fbClient := database.NewFirebaseClient(cfg)
	firebaseRepo := firebase.NewRepository(fbClient)

	// 3. Cache & Audit Engine Initialization
	// Strictly bounded 128 MB cache with SingleFlight stampede protection
	cacheEngine, err := cache.NewRistrettoCache(128, 100000)
	if err != nil {
		log.Fatalf("[FATAL] Failed to initialize Ristretto cache engine: %v", err)
	}
	defer cacheEngine.Close()

	auditor := audit.NewAuditor(5000)

	// 4. Service Layer Orchestration
	svc := services.NewService(
		activeStudentRepo,
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

	// 5. Handlers Initialization
	studentHandler := handlers.NewStudentAPIHandler(svc)
	academicsHandler := handlers.NewAcademicsAPIHandler(svc)
	attendanceHandler := handlers.NewAttendanceAPIHandler(svc)
	financeHandler := handlers.NewFinanceAPIHandler(svc)
	healthHandler := handlers.NewHealthLogisticsAPIHandler(svc)
	campusHandler := handlers.NewCampusAPIHandler(svc)
	ptmHandler := handlers.NewPtmAPIHandler(svc)
	auditStaffHandler := handlers.NewAuditStaffAPIHandler(svc, cfg.StorageMode)

	// 6. HTTP Router & Middleware Setup
	r := chi.NewRouter()

	r.Use(chimiddleware.RequestID)
	r.Use(chimiddleware.RealIP)
	r.Use(chimiddleware.Logger)
	r.Use(chimiddleware.Recoverer)
	r.Use(chimiddleware.Compress(5))
	r.Use(chimiddleware.Timeout(30 * time.Second))
	r.Use(appmiddleware.SecurityHeaders)
	r.Use(appmiddleware.MaxBodyLimit(10 * 1024 * 1024)) // 10MB limit
	r.Use(appmiddleware.NewCORS(cfg.CORSOrigins))

	// Rate limiter: 1,000 rps tolerant per IP
	rateLimiter := appmiddleware.NewRateLimiter(1000, 200)
	r.Use(rateLimiter.Middleware())

	// Top-level Health Checks
	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	})
	r.Get("/api/v1/health", auditStaffHandler.GetHealth)

	// 7. Route Directory: Mount All 64 Endpoints under /api/v1/
	r.Route("/api/v1", func(api chi.Router) {
		api.Use(appmiddleware.JWTAuth(cfg.JWTSecret))

		// Category 1: Student Identity, Admission & Profiles
		api.Get("/studentData", studentHandler.GetStudentData)
		api.Post("/newStudentData", studentHandler.NewStudentData)
		api.Put("/updateStudentData", studentHandler.UpdateStudentData)

		// Category 2: Academics, Time Tables & Classroom Operations
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

		// Category 3: Examinations, Report Cards & Results
		api.Post("/setExamDates", academicsHandler.SetExamDates)
		api.Get("/getExamDates", academicsHandler.GetExamDates)
		api.Get("/examTimeTable", academicsHandler.GetExamTimeTable)
		api.Get("/exam", academicsHandler.GetExamGuidelines)
		api.Get("/results", academicsHandler.GetResults)
		api.Post("/results", academicsHandler.BatchSaveResults)
		api.Get("/reportCard", academicsHandler.GetReportCard)

		// Category 4: Payments, Invoicing & GST Compliance
		api.Post("/payment", financeHandler.CreatePayment)
		api.Post("/payment/verify", financeHandler.VerifyPayment)
		api.Get("/paymentHistory", financeHandler.GetPaymentHistory)
		api.Post("/taxReceipt", financeHandler.GenerateTaxReceipt)

		// Category 5: Campus Logistics, Canteen, Store & Health
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

		// Category 6: Campus Life, Houses, Lost Property & Media
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

		// Category 7: Parent Desk, PTM & Notifications
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

		// Category 8: Staff Administration & Core System
		api.Get("/schoolInfo", auditStaffHandler.GetSchoolInfo)
		api.Post("/addStaff", auditStaffHandler.AddStaff)
		api.Put("/updateStaff", auditStaffHandler.UpdateStaff)
		api.Delete("/deleteStaff", auditStaffHandler.DeleteStaff)
	})

	// 8. Start HTTP Server with Tuned Settings
	addr := fmt.Sprintf(":%s", cfg.Port)
	srv := &http.Server{
		Addr:         addr,
		Handler:      r,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	serverErrors := make(chan error, 1)
	go func() {
		log.Printf("[INFO] NexaCampus Backend Engine running on port %s (http://0.0.0.0:%s)", cfg.Port, cfg.Port)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErrors <- err
		}
	}()

	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM, syscall.SIGINT)

	select {
	case err := <-serverErrors:
		log.Fatalf("[FATAL] Server failure: %v", err)
	case sig := <-shutdown:
		log.Printf("[INFO] Received signal %v, commencing graceful shutdown...", sig)
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := srv.Shutdown(ctx); err != nil {
			log.Printf("[ERROR] Graceful shutdown error: %v, closing immediately", err)
			_ = srv.Close()
		}
		log.Println("[INFO] NexaCampus Backend Engine stopped cleanly.")
	}
}

