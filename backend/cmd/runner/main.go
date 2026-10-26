package main

import (
	"fmt"
	"log"

	"rojgarbridge-backend/internal/config"
	"rojgarbridge-backend/internal/db"
	"rojgarbridge-backend/internal/extractor"
	"rojgarbridge-backend/internal/scrapers"
)

func main() {
	cfg := config.Load()
	database, err := db.InitDB(cfg.DBPath, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("db error: %v", err)
	}
	defer database.Close()

	ext := extractor.NewGeminiExtractor(cfg.GeminiAPIKey)
	mgr := scrapers.NewManager(database, ext)

	fmt.Println("1. Upgrading existing drafts with enhanced data extraction...")
	updated, errs := mgr.ReExtractDrafts()
	fmt.Printf("Upgraded: %d drafts (Errors: %d)\n", updated, errs)

	fmt.Println("2. Running full scraper pipeline...")
	res := mgr.RunPipeline()
	fmt.Printf("\nPipeline Completed Successfully!\nTotal Scanned: %d | New Drafts: %d | Skipped (Existing): %d | Errors: %d\n",
		res.TotalScanned, res.NewDrafts, res.Skipped, res.Errors)
}
