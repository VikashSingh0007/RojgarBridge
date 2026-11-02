package pipeline_test

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"rojgarbridge-backend/database"
	"rojgarbridge-backend/modules/job"
	"rojgarbridge-backend/modules/pipeline/extractor"
	"rojgarbridge-backend/modules/pipeline/scrapers"
)

// TestIBPSEndToEndExtraction verifies IBPS scraper and rule-based extractor
func TestIBPSEndToEndExtraction(t *testing.T) {
	// 1. Initialize IBPS scraper
	ibpsScraper := scrapers.NewIBPSScraper()
	if ibpsScraper.Name() != "IBPS" {
		t.Fatalf("Expected scraper name 'IBPS', got '%s'", ibpsScraper.Name())
	}
	t.Logf("Selected Scraper: %s (%s)", ibpsScraper.Name(), ibpsScraper.BaseURL())

	// 2. Fetch real raw notices from IBPS
	notices, err := ibpsScraper.FetchNotices()
	if err != nil || len(notices) == 0 {
		t.Logf("Live network fetch returned err: %v or empty; using realistic IBPS official notice fixture", err)
		notices = []job.RawNotice{
			{
				SourcePortal: "IBPS",
				NoticeTitle:  "Advertisement for Direct Recruitment of Server Administrator, Assistant & Programming Assistant on Regular Basis",
				NoticeURL:    "https://www.ibps.in/wp-content/uploads/Advertisement-for-Server-Administrator-Programming-Assistant-and-Assistant-Oct._Nov.-2026.pdf",
				PublishedAt:  "2026-10-29",
				FetchedAt:    time.Now(),
				ContentText: "Title: Advertisement for Direct Recruitment of Server Administrator, Assistant & Programming Assistant on Regular Basis | " +
					"Source: Institute of Banking Personnel Selection (Official Portal: ibps.in) | " +
					"Published Date: 2026-10-29 | " +
					"Official Link: https://www.ibps.in/wp-content/uploads/Advertisement-for-Server-Administrator-Programming-Assistant-and-Assistant-Oct._Nov.-2026.pdf",
			},
		}
	} else {
		t.Logf("Discovered %d raw notices from IBPS", len(notices))
	}

	// 3. Initialize rule-based extractor
	ext := extractor.NewRuleBasedExtractor()

	// 4. In-memory SQLite database for draft verification
	db, err := database.InitDB("file:memdb_ibps?mode=memory&cache=shared", "")
	if err != nil {
		t.Fatalf("Failed to initialize in-memory database: %v", err)
	}
	defer db.Close()

	jobService := job.NewService(db, nil)

	// 5. Select recruitment notice
	var sampleNotice job.RawNotice
	for _, n := range notices {
		if strings.Contains(strings.ToLower(n.NoticeTitle), "recruitment") || strings.Contains(strings.ToLower(n.NoticeTitle), "crp") {
			sampleNotice = n
			break
		}
	}
	if sampleNotice.NoticeTitle == "" {
		sampleNotice = notices[0]
	}

	t.Logf("\n=================== RAW IBPS NOTICE RECEIVED ===================")
	t.Logf("Source Portal: %s", sampleNotice.SourcePortal)
	t.Logf("Notice Title : %s", sampleNotice.NoticeTitle)
	t.Logf("Notice URL   : %s", sampleNotice.NoticeURL)
	t.Logf("Published At : %s", sampleNotice.PublishedAt)
	t.Logf("Content Text : %s", sampleNotice.ContentText)
	t.Logf("=================================================================")

	// 6. Extract structured details
	details, err := ext.Extract(sampleNotice)
	if err != nil {
		t.Fatalf("RuleBasedExtractor.Extract failed: %v", err)
	}

	detailsJSON, _ := json.MarshalIndent(details, "", "  ")
	t.Logf("\n============= EXTRACTED IBPS JOB DETAILS =============\n%s\n=======================================================", string(detailsJSON))

	// Assertions
	if details.Title == "" {
		t.Errorf("Expected non-empty Title")
	}
	if details.Organization != "Institute of Banking Personnel Selection (IBPS)" {
		t.Errorf("Expected Organization 'Institute of Banking Personnel Selection (IBPS)', got '%s'", details.Organization)
	}
	if details.Category != "Banking" {
		t.Errorf("Expected Category 'Banking', got '%s'", details.Category)
	}
	if details.State != "All India" {
		t.Errorf("Expected State 'All India', got '%s'", details.State)
	}
	if details.ConfidenceScore < 0.7 {
		t.Errorf("Expected ConfidenceScore >= 0.7, got %f", details.ConfidenceScore)
	}

	// 7. Verify draft persistence
	hash := scrapers.GenerateNoticeHash(sampleNotice.SourcePortal, sampleNotice.NoticeTitle, sampleNotice.NoticeURL)
	draftJob := &job.Job{
		Hash:         hash,
		SourcePortal: sampleNotice.SourcePortal,
		Status:       job.StatusDraft,
		RawNotice:    sampleNotice,
		Details:      *details,
		ReviewNotes:  "Auto-scraped via IBPS verified extraction",
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	if err := jobService.SaveDraft(draftJob); err != nil {
		t.Fatalf("Failed to save IBPS draft to database: %v", err)
	}
	t.Logf("Successfully inserted IBPS draft into database with ID: %d", draftJob.ID)
}
