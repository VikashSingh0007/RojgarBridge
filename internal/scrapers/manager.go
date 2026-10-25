package scrapers

import (
	"log"
	"sync"
	"time"

	"rojgarbridge-backend/internal/db"
	"rojgarbridge-backend/internal/extractor"
	"rojgarbridge-backend/internal/models"
)

type PipelineResult struct {
	TotalScanned int `json:"total_scanned"`
	NewDrafts    int `json:"new_drafts"`
	Skipped      int `json:"skipped"`
	Errors       int `json:"errors"`
}

type Manager struct {
	scrapers  []PortalScraper
	extractor extractor.Extractor
	db        *db.DB
}

func NewManager(database *db.DB, ext extractor.Extractor) *Manager {
	m := &Manager{
		db:        database,
		extractor: ext,
	}

	// 1. Direct Government Portals
	m.RegisterScraper(NewSSCScraper())
	m.RegisterScraper(NewUPSCScraper())

	// 2. High-Volume Specialized Category Table Scrapers (0 = unlimited, scrapes all jobs on page)
	m.RegisterScraper(NewCategoryTableScraper("Banking Jobs", "https://www.freejobalert.com/bank-jobs/", "Banking", 0))
	m.RegisterScraper(NewCategoryTableScraper("Railway Jobs", "https://www.freejobalert.com/railway-jobs/", "Railway", 0))
	m.RegisterScraper(NewCategoryTableScraper("Police & Defence Jobs", "https://www.freejobalert.com/police-defence-jobs/", "Defence & Police", 0))
	m.RegisterScraper(NewCategoryTableScraper("Teaching Jobs", "https://www.freejobalert.com/teaching-faculty-jobs/", "Teaching", 0))
	m.RegisterScraper(NewCategoryTableScraper("Engineering Jobs", "https://www.freejobalert.com/engineering-jobs/", "Engineering & PSU", 0))
	m.RegisterScraper(NewCategoryTableScraper("State Govt Jobs", "https://www.freejobalert.com/state-government-jobs/", "State Govt", 0))
	m.RegisterScraper(NewCategoryTableScraper("Central Govt Jobs", "https://www.freejobalert.com/government-jobs/", "Central Govt", 0))

	// 3. Exam Updates (Admit Cards & Results)
	m.RegisterScraper(NewCategoryTableScraper("Admit Cards", "https://www.freejobalert.com/admit-card/", "Admit Card", 0))
	m.RegisterScraper(NewCategoryTableScraper("Exam Results", "https://www.freejobalert.com/exam-results/", "Result", 0))

	// 4. Real-time RSS Feed Scraper
	m.RegisterScraper(NewRSSScraper("Live Job Alerts Feed", "https://www.freejobalert.com/feed"))

	return m
}

func (m *Manager) RegisterScraper(s PortalScraper) {
	m.scrapers = append(m.scrapers, s)
}

// RunPipeline executes all scrapers, extracts with AI/heuristics, and stores as drafts
func (m *Manager) RunPipeline() PipelineResult {
	var result PipelineResult
	var mu sync.Mutex

	var wg sync.WaitGroup

	for _, scraper := range m.scrapers {
		wg.Add(1)
		go func(s PortalScraper) {
			defer wg.Done()

			log.Printf("[Scraper] Running %s scraper...", s.Name())
			notices, err := s.FetchNotices()
			if err != nil {
				log.Printf("[Scraper Error] %s: %v", s.Name(), err)
				mu.Lock()
				result.Errors++
				mu.Unlock()
				return
			}

			log.Printf("[Scraper] %s found %d raw notices", s.Name(), len(notices))

			for _, notice := range notices {
				mu.Lock()
				result.TotalScanned++
				mu.Unlock()

				hash := GenerateNoticeHash(notice.SourcePortal, notice.NoticeTitle, notice.NoticeURL)
				exists, err := m.db.ExistsByHash(hash)
				if err != nil {
					log.Printf("[DB Error] Exists check failed: %v", err)
					continue
				}

				if exists {
					mu.Lock()
					result.Skipped++
					mu.Unlock()
					continue
				}

				// Extract structured details
				details, err := m.extractor.Extract(notice)
				if err != nil {
					log.Printf("[Extractor Error] Failed for %s: %v", notice.NoticeTitle, err)
					mu.Lock()
					result.Errors++
					mu.Unlock()
					continue
				}

				job := &models.Job{
					Hash:         hash,
					SourcePortal: notice.SourcePortal,
					Status:       models.StatusDraft,
					RawNotice:    notice,
					Details:      *details,
					ReviewNotes:  "Auto-scraped with enriched metadata. Awaiting review.",
					CreatedAt:    time.Now(),
					UpdatedAt:    time.Now(),
				}

				if err := m.db.SaveDraft(job); err != nil {
					log.Printf("[DB Error] Failed to save draft: %v", err)
					mu.Lock()
					result.Errors++
					mu.Unlock()
				} else {
					log.Printf("[Draft Created] ID: %d | [%s] %s (Vacancies: %d, Last Date: %s)",
						job.ID, job.Details.Category, job.Details.Title, job.Details.TotalVacancies, job.Details.ApplicationEnd)
					mu.Lock()
					result.NewDrafts++
					mu.Unlock()
				}
			}
		}(scraper)
	}

	wg.Wait()
	return result
}

// ReExtractDrafts upgrades all existing drafts in database with enhanced extraction
func (m *Manager) ReExtractDrafts() (int, int) {
	jobs, err := m.db.GetJobsForReExtraction()
	if err != nil {
		log.Printf("[ReExtract Error] Failed to fetch drafts: %v", err)
		return 0, 0
	}

	updatedCount := 0
	errorCount := 0

	for _, job := range jobs {
		details, err := m.extractor.Extract(job.RawNotice)
		if err != nil {
			errorCount++
			continue
		}

		job.Details = *details
		job.ReviewNotes = "Upgraded with enhanced heuristic data extractor"
		if err := m.db.UpdateJob(&job); err != nil {
			errorCount++
		} else {
			updatedCount++
		}
	}

	log.Printf("[ReExtract] Finished upgrading existing drafts: %d updated, %d errors", updatedCount, errorCount)
	return updatedCount, errorCount
}
