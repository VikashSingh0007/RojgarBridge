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

	"rojgarbridge-backend/internal/api"
	"rojgarbridge-backend/internal/config"
	"rojgarbridge-backend/internal/db"
	"rojgarbridge-backend/internal/extractor"
	"rojgarbridge-backend/internal/notifier"
	"rojgarbridge-backend/internal/scrapers"
)

func main() {
	cfg := config.Load()
	log.Println("==================================================")
	log.Println("      RojgarBridge Backend Engine Starting        ")
	log.Println("==================================================")

	// 1. Initialize Database (Neon Postgres or fallback SQLite)
	database, err := db.InitDB(cfg.DBPath, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Fatal: Database initialization failed: %v", err)
	}
	defer database.Close()
	if cfg.DatabaseURL != "" {
		log.Println("[DB] Connected to Neon PostgreSQL database.")
	} else {
		log.Printf("[DB] Connected to SQLite database: %s", cfg.DBPath)
	}

	// 2. Initialize Extractor
	if cfg.GeminiAPIKey != "" {
		log.Println("[AI] Gemini AI Extractor enabled with API key.")
	} else {
		log.Println("[AI] GEMINI_API_KEY not set. Using smart rule-based fallback extractor.")
	}
	aiExtractor := extractor.NewGeminiExtractor(cfg.GeminiAPIKey)

	// 3. Initialize Scrapers Manager
	manager := scrapers.NewManager(database, aiExtractor)

	// 4. Background scheduler (runs pipeline every 6 hours or on start)
	go func() {
		log.Println("[Scheduler] Running initial scraping pass...")
		res := manager.RunPipeline()
		log.Printf("[Scheduler] Initial pass completed: Scanned: %d, New Drafts: %d, Skipped: %d, Errors: %d",
			res.TotalScanned, res.NewDrafts, res.Skipped, res.Errors)

		ticker := time.NewTicker(6 * time.Hour)
		defer ticker.Stop()
		for range ticker.C {
			log.Println("[Scheduler] Running periodic scraper pass...")
			manager.RunPipeline()
		}
	}()

	// 5. Initialize Telegram Alert Notifier
	tgNotifier := notifier.NewTelegramNotifier(cfg.TelegramBotToken, cfg.TelegramChatID)
	if tgNotifier.IsEnabled() {
		log.Println("[Notifier] Telegram Channel Alert Broadcast enabled.")
	}

	// 6. Initialize API Router
	handler := api.NewHandler(database, manager, tgNotifier, cfg)
	router := api.NewRouter(handler)

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
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Printf("Server forced shutdown: %v", err)
	}
	log.Println("[Server] RojgarBridge backend exited clean.")
	fmt.Println("Bye!")
}
