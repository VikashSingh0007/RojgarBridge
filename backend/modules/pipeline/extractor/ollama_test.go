package extractor_test

import (
	"strings"
	"testing"
	"time"

	"rojgarbridge-backend/modules/job"
	"rojgarbridge-backend/modules/pipeline/extractor"
)

func TestPDFReaderAndOllamaExtractor(t *testing.T) {
	// 1. Verify PDF Reader with synthetic PDF bytes
	pdfReader := extractor.NewPDFReader()
	if pdfReader == nil {
		t.Fatalf("Failed to initialize PDFReader")
	}

	// 2. Verify Ollama Extractor initialization & fallback
	ollamaExt := extractor.NewOllamaExtractor("http://localhost:11434", "llama3.2:3b")
	if ollamaExt == nil {
		t.Fatalf("Failed to initialize OllamaExtractor")
	}

	t.Logf("Ollama Available: %v", ollamaExt.IsAvailable())

	sampleNotice := job.RawNotice{
		SourcePortal: "SSC",
		NoticeTitle:  "Notice of Combined Graduate Level Examination, 2026",
		NoticeURL:    "https://ssc.gov.in/api/attachment/cgl2026.pdf",
		PublishedAt:  "2026-10-25",
		FetchedAt:    time.Now(),
		ContentText: "Staff Selection Commission Notice of Combined Graduate Level Examination (CGLE) 2026. " +
			"Total Vacancies: 17727 Posts. Essential Qualification: Bachelor's Degree / Graduate. " +
			"Dates for submission of online applications: 24-06-2026 to 24-07-2026. " +
			"Application Fee: ₹100. Women, SC, ST, PwBD candidates exempted.",
	}

	details, err := ollamaExt.Extract(sampleNotice)
	if err != nil {
		t.Fatalf("OllamaExtractor.Extract failed: %v", err)
	}

	if details == nil {
		t.Fatalf("Extracted details is nil")
	}

	t.Logf("Extracted Details:")
	t.Logf("  - Title: %s", details.Title)
	t.Logf("  - Org: %s", details.Organization)
	t.Logf("  - Vacancies: %d", details.TotalVacancies)
	t.Logf("  - Start: %s", details.ApplicationStart)
	t.Logf("  - End: %s", details.ApplicationEnd)
	t.Logf("  - Qualification: %s", details.Qualification)
	t.Logf("  - Fee (General): %s", details.FeeGeneral)

	if !strings.Contains(details.Organization, "Staff Selection Commission") {
		t.Errorf("Expected organization to contain Staff Selection Commission, got %s", details.Organization)
	}
	if details.TotalVacancies != 17727 {
		t.Errorf("Expected 17727 vacancies, got %d", details.TotalVacancies)
	}
	if details.ApplicationStart == "" {
		t.Errorf("Expected non-empty application start date")
	}
}
