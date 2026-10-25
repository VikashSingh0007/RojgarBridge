package scrapers

import (
	"log"
	"sync"
	"time"

	"rozgarhub-backend/internal/db"
	"rozgarhub-backend/internal/extractor"
	"rozgarhub-backend/internal/models"
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

	// Register default scrapers
	m.RegisterScraper(NewSSCScraper())
	m.RegisterScraper(NewUPSCScraper())
	m.RegisterScraper(NewRSSScraper("Central Govt Jobs", "https://www.freejobalert.com/feed"))

	return m
}

func (m *Manager) RegisterScraper(s PortalScraper) {
	m.scrapers = append(m.scrapers, s)
}

// RunPipeline executes all scrapers, extracts with AI, and stores as drafts
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

				// New notice! Extract details with AI
				log.Printf("[AI Extractor] Processing notice: %s (%s)", notice.NoticeTitle, notice.SourcePortal)
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
					ReviewNotes:  "Auto-scraped and extracted by AI. Awaiting review.",
					CreatedAt:    time.Now(),
					UpdatedAt:    time.Now(),
				}

				if err := m.db.SaveDraft(job); err != nil {
					log.Printf("[DB Error] Failed to save draft: %v", err)
					mu.Lock()
					result.Errors++
					mu.Unlock()
				} else {
					log.Printf("[Draft Created] ID: %d | Title: %s", job.ID, job.Details.Title)
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
