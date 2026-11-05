package main

import (
	"flag"
	"fmt"
	"log"

	"rojgarbridge-backend/config"
	"rojgarbridge-backend/database"
	"rojgarbridge-backend/modules/job"
	"rojgarbridge-backend/modules/pipeline"
	"rojgarbridge-backend/modules/pipeline/extractor"
)

func main() {
	portalFlag := flag.String("portal", "", "Specify a single portal scraper to run (e.g. 'ssc')")
	cleanFlag := flag.Bool("clean", false, "Clean existing stale jobs before fresh scrape")
	flag.Parse()

	cfg := config.Load()
	dbConn, err := database.InitDB(cfg.DBPath, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("db error: %v", err)
	}
	defer dbConn.Close()

	if *cleanFlag {
		fmt.Println("Clearing stale jobs from database for fresh extraction...")
		if _, err := dbConn.Exec("DELETE FROM jobs"); err != nil {
			log.Fatalf("Failed to clear jobs: %v", err)
		}
		fmt.Println("Jobs table cleared cleanly.")
	}

	ext := extractor.NewOllamaExtractor("", "llama3.2:3b")
	jobSvc := job.NewService(dbConn, nil)
	pipeSvc := pipeline.NewService(jobSvc, ext)

	if *portalFlag != "" {
		fmt.Printf("Running scraper batch for portal: %s...\n", *portalFlag)
		res := pipeSvc.RunScraper(*portalFlag)
		fmt.Printf("\nBatch Completed for %s!\nTotal Scanned: %d | New Drafts: %d | Skipped (Existing): %d | Errors: %d\n",
			*portalFlag, res.TotalScanned, res.NewDrafts, res.Skipped, res.Errors)
		return
	}

	fmt.Println("1. Upgrading existing drafts with enhanced data extraction...")
	updated, errs := pipeSvc.ReExtractDrafts()
	fmt.Printf("Upgraded: %d drafts (Errors: %d)\n", updated, errs)

	fmt.Println("2. Running full scraper pipeline...")
	res := pipeSvc.RunPipeline()
	fmt.Printf("\nPipeline Completed Successfully!\nTotal Scanned: %d | New Drafts: %d | Skipped (Existing): %d | Errors: %d\n",
		res.TotalScanned, res.NewDrafts, res.Skipped, res.Errors)
}
