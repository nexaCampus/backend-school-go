package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/nexaCampus/backend-school-go/internal/config"
	"github.com/nexaCampus/backend-school-go/internal/database"
	"github.com/nexaCampus/backend-school-go/internal/handlers"
	appmiddleware "github.com/nexaCampus/backend-school-go/internal/middleware"
)

func main() {
	log.Println("[INFO] Bootstrapping Stitch School Portal Backend...")

	// 1. Load configuration
	cfg := config.Load()

	// 2. Initialize database connection pool
	pool, err := database.InitDB(cfg)
	if err != nil {
		log.Fatalf("[FATAL] Could not initialize database: %v", err)
	}
	defer pool.Close()

	// 3. Instantiate handlers covering all 21 modules
	authHandler := handlers.NewAuthHandler(pool, cfg.JWTSecret)
	studentHandler := handlers.NewStudentHandler(pool, cfg.JWTSecret)
	homeworkHandler := handlers.NewHomeworkHandler(pool)
	attendanceHandler := handlers.NewAttendanceHandler(pool)
	timetableHandler := handlers.NewTimetableHandler(pool)
	feeHandler := handlers.NewFeeHandler(pool)
	noticeHandler := handlers.NewNoticeHandler(pool)
	helpdeskHandler := handlers.NewHelpdeskHandler(pool)
	transportHandler := handlers.NewTransportHandler(pool)
	miscHandler := handlers.NewMiscHandler(pool)
	academicsHandler := handlers.NewAcademicsHandler(pool)
	ptmHandler := handlers.NewPtmHandler(pool)
	hallPassHandler := handlers.NewHallPassHandler(pool)
	healthHandler := handlers.NewHealthHandler(pool)
	meritsHandler := handlers.NewMeritsHandler(pool)
	storeHandler := handlers.NewStoreHandler(pool)
	lostFoundHandler := handlers.NewLostFoundHandler(pool)
	galleryHandler := handlers.NewGalleryHandler(pool)
	clubsHandler := handlers.NewClubsHandler(pool)

	// 4. Setup Router and Middlewares
	r := chi.NewRouter()

	r.Use(chimiddleware.RequestID)
	r.Use(chimiddleware.RealIP)
	r.Use(chimiddleware.Logger)
	r.Use(chimiddleware.Recoverer)
	r.Use(chimiddleware.Compress(5))
	r.Use(chimiddleware.Timeout(60 * time.Second))
	r.Use(appmiddleware.NewCORS(cfg.CORSOrigins))

	// Health check for Render uptime ping monitor
	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	})

	// 5. Mount API Routes under /v1
	r.Route("/v1", func(v1 chi.Router) {
		// 1. System & Authentication
		v1.Post("/auth/verify", authHandler.Verify)
		v1.Post("/auth/refresh", authHandler.Refresh)

		// Portal group (supports optional/bearer JWT)
		v1.Group(func(portal chi.Router) {
			portal.Use(appmiddleware.OptionalAuthMiddleware(cfg.JWTSecret))

			// 2. Student Profile & Identity
			portal.Get("/students/{id}", studentHandler.GetProfile)
			portal.Put("/students/{id}/avatar", studentHandler.UpdateAvatar)
			portal.Get("/students/{id}/digital-badge", studentHandler.GetDigitalBadge)

			// 3. Home Dashboard
			portal.Get("/dashboard/summary", studentHandler.GetDashboardSummary)

			// 4. Homework & Diary
			portal.Get("/homework", homeworkHandler.ListHomework)
			portal.Get("/homework/{id}", homeworkHandler.GetHomeworkDetail)
			portal.Post("/homework/{id}/submit", homeworkHandler.SubmitHomework)

			// 5. Attendance & Leave Applications
			portal.Get("/attendance", attendanceHandler.GetMonthlyAttendance)
			portal.Get("/attendance/summary", attendanceHandler.GetAttendanceSummary)
			portal.Post("/attendance/leave", attendanceHandler.SubmitLeave)
			portal.Get("/attendance/leave", attendanceHandler.ListLeaveRequests)

			// 6. Timetable & Bell Schedule
			portal.Get("/timetable", timetableHandler.GetTimetable)

			// 7. Fees & Academic Performance
			portal.Get("/fees", feeHandler.GetFees)
			portal.Get("/academics/report-card", academicsHandler.GetReportCard)

			// 8. Notice Board & Circulars
			portal.Get("/notices", noticeHandler.ListNotices)

			// 9. Helpdesk & Support
			portal.Post("/helpdesk/tickets", helpdeskHandler.CreateTicket)
			portal.Get("/helpdesk/tickets", helpdeskHandler.ListTickets)

			// 10. School Transport & Live GPS
			portal.Get("/transport/routes/{route}", transportHandler.GetRouteDetails)
			portal.Post("/transport/routes/{route}/location", transportHandler.UpdateLocation)

			// 11. School Library
			portal.Get("/library/books", miscHandler.SearchBooks)
			portal.Get("/library/borrowings", miscHandler.GetBorrowings)
			portal.Post("/library/reserve", miscHandler.ReserveBook)

			// 12. Exam Hall Tickets
			portal.Get("/exams/hall-ticket", miscHandler.GetHallTicket)

			// 13. Canteen & Smart Card Wallet
			portal.Get("/canteen/wallet", miscHandler.GetCanteenWallet)
			portal.Get("/canteen/menu", miscHandler.GetCanteenMenu)

			// 14. Parent-Teacher Meeting (PTM) Scheduler
			portal.Get("/ptm/sessions", ptmHandler.ListSessions)
			portal.Get("/ptm/slots", ptmHandler.ListSlots)
			portal.Post("/ptm/book", ptmHandler.BookSlot)

			// 15. Digital Hall Pass Engine
			portal.Post("/hallpass/request", hallPassHandler.RequestPass)
			portal.Get("/hallpass/active", hallPassHandler.GetActivePass)

			// 16. Student Infirmary & Medical Dashboard
			portal.Get("/health/profile", healthHandler.GetProfile)
			portal.Get("/health/clinic-visits", healthHandler.GetClinicVisits)
			portal.Post("/health/consent", healthHandler.SubmitConsent)

			// 17. House System, Merits & Discipline
			portal.Get("/merits/summary", meritsHandler.GetSummary)
			portal.Get("/merits/records", meritsHandler.GetRecords)

			// 18. School Store & Inventory
			portal.Get("/store/products", storeHandler.ListProducts)
			portal.Post("/store/order", storeHandler.CreateOrder)

			// 19. Campus Lost & Found
			portal.Get("/lost-found/items", lostFoundHandler.ListItems)
			portal.Post("/lost-found/report", lostFoundHandler.ReportItem)

			// 20. Event Gallery & Yearbook
			portal.Get("/gallery/albums", galleryHandler.ListAlbums)
			portal.Get("/gallery/albums/{id}", galleryHandler.GetAlbumPhotos)

			// 21. Clubs & Societies
			portal.Get("/clubs/enrolled", clubsHandler.ListEnrolled)
			portal.Get("/clubs/competitions", clubsHandler.ListCompetitions)
			portal.Post("/clubs/competitions/{id}/register", clubsHandler.RegisterCompetition)
		})
	})

	// 6. Start HTTP Server with Graceful Shutdown
	addr := fmt.Sprintf(":%s", cfg.Port)
	srv := &http.Server{
		Addr:         addr,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	serverErrors := make(chan error, 1)
	go func() {
		log.Printf("[INFO] Server listening on port %s (http://0.0.0.0:%s)", cfg.Port, cfg.Port)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErrors <- err
		}
	}()

	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM, syscall.SIGINT)

	select {
	case err := <-serverErrors:
		log.Fatalf("[FATAL] Server runtime failure: %v", err)
	case sig := <-shutdown:
		log.Printf("[INFO] Received signal %v, initiating graceful shutdown...", sig)
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := srv.Shutdown(ctx); err != nil {
			log.Printf("[ERROR] Graceful shutdown failed: %v, forcing close", err)
			_ = srv.Close()
		}
		log.Println("[INFO] Server stopped gracefully.")
	}
}
