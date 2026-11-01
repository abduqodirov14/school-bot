package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"

	"school-attendance/internal/api"
	"school-attendance/internal/bot"
	"school-attendance/internal/config"
	"school-attendance/internal/db"
	"school-attendance/internal/importer"
	"school-attendance/internal/scheduler"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Konfiguratsiyani yuklashda xatolik: %v", err)
	}

	log.Printf("Maktab Davomati tizimi ishga tushmoqda...")
	log.Printf("Vaqt zonasi: %s | Port: %s", cfg.Timezone, cfg.Port)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 1. Database connection & migrations
	database, err := db.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Printf("Ogohlantirish: Ma'lumotlar bazasiga ulanib bo'lmadi (%v).", err)
	} else {
		log.Printf("Ma'lumotlar bazasi va migratsiyalar muvaffaqiyatli ishga tushirildi.")
		defer database.Pool.Close()
	}

	// 2. Telegram Bot initialization
	var tgBot *bot.Bot
	if cfg.BotToken != "" && database != nil {
		var botErr error
		tgBot, botErr = bot.New(cfg, database)
		if botErr != nil {
			log.Printf("Telegram botni ishga tushirishda xatolik: %v", botErr)
		} else {
			go tgBot.Start(ctx)
		}
	} else {
		log.Println("Eslatma: BOT_TOKEN yoki ma'lumotlar bazasi ulanmaganligi sababli bot ishga tushmadi.")
	}

	// 3. Scheduler initialization (Cron in Asia/Tashkent)
	var cronScheduler *scheduler.Scheduler
	if database != nil {
		cronScheduler = scheduler.New(cfg, database, tgBot)
		cronScheduler.Start(ctx)
		defer cronScheduler.Stop()
	}

	// 4. HTTP Router setup
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(60 * time.Second))

	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-Telegram-Init-Data"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	// Public routes
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok","time":"` + time.Now().In(cfg.Location).Format(time.RFC3339) + `"}`))
	})

	// API routes
	if database != nil {
		authHandler := api.NewAuthHandler(database)
		classHandler := api.NewClassHandler(database, cfg)
		studentHandler := api.NewStudentHandler(database)
		attendanceHandler := api.NewAttendanceHandler(database, cfg)
		dashboardHandler := api.NewDashboardHandler(database, cfg)
		reportHandler := api.NewReportHandler(database, cfg)

		importerService := importer.NewService(database)
		_ = importerService.MarkStaleJobsFailed(context.Background())
		importHandler := api.NewImportHandler(importerService)

		if cronScheduler != nil {
			attendanceHandler.SetOnAttendanceSaved(func(studentID int64, schoolID int64, date time.Time) {
				cronScheduler.CheckSingleStudentThreshold(context.Background(), studentID, schoolID, date)
			})
		}

		r.Route("/api", func(apiRouter chi.Router) {
			// Public template download
			apiRouter.Get("/import/template", importHandler.GetTemplate)

			// Protected routes with Telegram initData validation
			apiRouter.Group(func(protected chi.Router) {
				protected.Use(api.AuthMiddleware(cfg.BotToken, database))

				// User profile
				protected.Get("/auth/me", authHandler.GetMe)

				// Dashboard (Director)
				protected.With(api.RequireRole("director")).Get("/dashboard/today", dashboardHandler.GetTodayDashboard)
				protected.With(api.RequireRole("director")).Get("/settings", dashboardHandler.GetSettings)
				protected.With(api.RequireRole("director")).Put("/settings", dashboardHandler.UpdateSettings)

				// Classes
				protected.Get("/classes", classHandler.ListClasses)
				protected.With(api.RequireRole("director")).Post("/classes", classHandler.CreateClass)
				protected.With(api.RequireRole("director")).Post("/classes/bulk", classHandler.BulkCreateClasses)
				protected.With(api.RequireRole("director")).Put("/classes/{id}", classHandler.UpdateClass)
				protected.With(api.RequireRole("director")).Delete("/classes/{id}", classHandler.DeleteClass)
				protected.With(api.RequireRole("director")).Get("/teachers", classHandler.ListTeachers)

				// Class Teachers & Substitutes (Director)
				protected.With(api.RequireRole("director")).Get("/classes/teachers/overview", classHandler.ListTeachersOverview)
				protected.With(api.RequireRole("director")).Post("/classes/{id}/teacher", classHandler.AssignTeacher)
				protected.With(api.RequireRole("director")).Delete("/classes/{id}/teacher", classHandler.UnassignTeacher)
				protected.With(api.RequireRole("director")).Post("/classes/{id}/substitute", classHandler.AssignSubstitute)

				// Import (Director)
				protected.With(api.RequireRole("director")).Post("/import/preview", importHandler.Preview)
				protected.With(api.RequireRole("director")).Post("/import/{job_id}/commit", importHandler.Commit)
				protected.With(api.RequireRole("director")).Get("/import/{job_id}", importHandler.GetStatus)
				protected.With(api.RequireRole("director")).Get("/import/{job_id}/errors/download", importHandler.DownloadErrors)
				protected.With(api.RequireRole("director")).Post("/import/deactivate-missing", importHandler.DeactivateMissing)

				// Students
				protected.Get("/students", studentHandler.ListStudents)
				protected.Get("/students/{id}", studentHandler.GetStudent)
				protected.Get("/students/{id}/history", dashboardHandler.GetStudentHistory)
				protected.With(api.RequireRole("director")).Post("/students", studentHandler.CreateStudent)
				protected.With(api.RequireRole("director")).Put("/students/{id}", studentHandler.UpdateStudent)
				protected.With(api.RequireRole("director")).Delete("/students/{id}", studentHandler.DeleteStudent)
				protected.With(api.RequireRole("director")).Post("/students/import", studentHandler.ImportStudentsExcel)

				// Attendance
				protected.Get("/attendance", attendanceHandler.GetAttendance)
				protected.Post("/attendance", attendanceHandler.SaveAttendance)
				protected.Get("/attendance/logs", attendanceHandler.GetAttendanceLogs)

				// Excel Reports (Director)
				protected.With(api.RequireRole("director")).Get("/reports/monthly", reportHandler.ExportMonthlyExcel)
			})
		})
	}

	// Serve frontend SPA (Telegram Mini App) from web/dist
	workDir, _ := os.Getwd()
	distDir := filepath.Join(workDir, "web", "dist")
	if _, err := os.Stat(distDir); err == nil {
		fileServer := http.FileServer(http.Dir(distDir))
		r.Get("/*", func(w http.ResponseWriter, r *http.Request) {
			requestedPath := filepath.Join(distDir, filepath.Clean(r.URL.Path))
			if info, err := os.Stat(requestedPath); err == nil && !info.IsDir() {
				fileServer.ServeHTTP(w, r)
				return
			}
			http.ServeFile(w, r, filepath.Join(distDir, "index.html"))
		})
		log.Printf("Frontend 'web/dist' statik fayllari ulandi.")
	}

	server := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		log.Printf("Server :%s portida tinglamoqda", cfg.Port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server xatosi: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Server to'xtatilmoqda...")
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("Serverni to'xtatishda xatolik: %v", err)
	}
	log.Println("Server to'xtatildi.")
}
