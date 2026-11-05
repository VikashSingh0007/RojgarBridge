package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"rojgarbridge-backend/config"
	"rojgarbridge-backend/database"
	"rojgarbridge-backend/modules/auth"
	"rojgarbridge-backend/modules/job"
	"rojgarbridge-backend/modules/pipeline"
	"rojgarbridge-backend/modules/pipeline/extractor"
	"rojgarbridge-backend/modules/user"
	"rojgarbridge-backend/utils"
)

func main() {
	cfg := config.Load()
	log.Println("==================================================")
	log.Println("      RojgarBridge Backend Engine Starting        ")
	log.Println("==================================================")

	// 1. Initialize Database (Neon Postgres or fallback SQLite)
	dbConn, err := database.InitDB(cfg.DBPath, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Fatal: Database initialization failed: %v", err)
	}
	defer dbConn.Close()
	if cfg.DatabaseURL != "" {
		log.Println("[DB] Connected to Neon PostgreSQL database.")
	} else {
		log.Printf("[DB] Connected to SQLite database: %s", cfg.DBPath)
	}

	// 2. Initialize Shared Utilities (Cache & Telegram Notifier)
	appCache := utils.NewCache(2 * time.Minute)
	tgNotifier := utils.NewTelegramNotifier(cfg.TelegramBotToken, cfg.TelegramChatID)
	if tgNotifier.IsEnabled() {
		log.Println("[Notifier] Telegram Channel Alert Broadcast enabled.")
	}

	// 3. Initialize Extractor (Ollama Local LLM + PDF Reader with deterministic Rule-Based Fallback)
	log.Println("[Extractor] Initializing Ollama local LLM + PDF extractor with Rule-Based Fallback...")
	ruleExtractor := extractor.NewOllamaExtractor("", "llama3.2:3b")

	// 4. Initialize Domain Services (with Dependency Injection)
	userService := user.NewService(dbConn)
	if err := userService.EnsureDefaultAdmin(); err != nil {
		log.Printf("[Warning] Failed ensuring default admin: %v", err)
	}

	jobService := job.NewService(dbConn, tgNotifier)
	if err := jobService.EnsurePublishedJobs(); err != nil {
		log.Printf("[Warning] Failed ensuring published jobs: %v", err)
	}

	authService := auth.NewService(userService, cfg)
	pipelineService := pipeline.NewService(jobService, ruleExtractor)

	// 5. Centralized Cron & Background Processors Startup
	procCtx, procCancel := context.WithCancel(context.Background())
	defer procCancel()

	userProcessor := user.NewProcessor(userService)
	authProcessor := auth.NewProcessor(authService)
	jobProcessor := job.NewProcessor(jobService)
	pipelineProcessor := pipeline.NewProcessor(pipelineService, 6*time.Hour)

	_ = userProcessor.Start(procCtx)
	_ = authProcessor.Start(procCtx)
	_ = jobProcessor.Start(procCtx)
	if err := pipelineProcessor.Start(procCtx); err != nil {
		log.Printf("[Warning] Failed starting pipeline processor: %v", err)
	}

	// 6. Initialize Module Handlers
	userHandler := user.NewHandler(userService)
	authHandler := auth.NewHandler(authService, cfg)
	jobHandler := job.NewHandler(jobService, appCache)
	pipelineHandler := pipeline.NewHandler(pipelineService)

	// 7. Centralized Routing
	router := NewRouter(ModuleRouters{
		AuthHandler:     authHandler,
		UserHandler:     userHandler,
		JobHandler:      jobHandler,
		PipelineHandler: pipelineHandler,
	})

	server := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Server runner
	go func() {
		log.Printf("[HTTP] RojgarBridge server listening on http://localhost:%s", cfg.Port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("HTTP server error: %v", err)
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("[Server] Shutting down gracefully...")
	procCancel()
	_ = pipelineProcessor.Stop()
	_ = jobProcessor.Stop()
	_ = authProcessor.Stop()
	_ = userProcessor.Stop()

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("Server forced shutdown: %v", err)
	}
	log.Println("[Server] RojgarBridge backend exited clean.")
	fmt.Println("Bye!")
}
