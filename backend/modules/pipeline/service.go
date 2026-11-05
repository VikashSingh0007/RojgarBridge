package pipeline

import (
	"log"
	"strings"
	"sync"
	"time"

	"rojgarbridge-backend/modules/job"
	"rojgarbridge-backend/modules/pipeline/extractor"
	"rojgarbridge-backend/modules/pipeline/scrapers"
)

type Service interface {
	RegisterScraper(s scrapers.PortalScraper)
	RunPipeline() PipelineResult
	RunScraper(name string) PipelineResult
	ReExtractDrafts() (int, int)
}

type service struct {
	scrapers   []scrapers.PortalScraper
	extractor  extractor.Extractor
	jobService job.Service
}

func NewService(jobService job.Service, ext extractor.Extractor) Service {
	s := &service{
		jobService: jobService,
		extractor:  ext,
	}

	// Register 100% Direct Official Government Portals ONLY
	s.RegisterScraper(scrapers.NewSSCScraper())   // Staff Selection Commission (ssc.gov.in)
	s.RegisterScraper(scrapers.NewUPSCScraper())  // Union Public Service Commission (upsc.gov.in)
	s.RegisterScraper(scrapers.NewIBPSScraper())  // Institute of Banking Personnel Selection (ibps.in)
	s.RegisterScraper(scrapers.NewBPSCScraper())  // Bihar Public Service Commission (bpsc.bihar.gov.in)
	s.RegisterScraper(scrapers.NewDSSSBScraper()) // Delhi Subordinate Services Selection Board (dsssb.delhi.gov.in)
	s.RegisterScraper(scrapers.NewNTAScraper())   // National Testing Agency (nta.ac.in)
	s.RegisterScraper(scrapers.NewUPPSCScraper()) // Uttar Pradesh Public Service Commission (uppsc.up.nic.in)
	s.RegisterScraper(scrapers.NewISROScraper())  // Indian Space Research Organisation (isro.gov.in)

	return s
}

func (s *service) RegisterScraper(sc scrapers.PortalScraper) {
	s.scrapers = append(s.scrapers, sc)
}

func (s *service) RunPipeline() PipelineResult {
	var result PipelineResult
	var mu sync.Mutex
	var wg sync.WaitGroup

	for _, scraper := range s.scrapers {
		wg.Add(1)
		go func(sc scrapers.PortalScraper) {
			defer wg.Done()

			log.Printf("[Scraper] Running %s scraper...", sc.Name())
			notices, err := sc.FetchNotices()
			if err != nil {
				log.Printf("[Scraper Error] %s: %v", sc.Name(), err)
				mu.Lock()
				result.Errors++
				mu.Unlock()
				return
			}

			log.Printf("[Scraper] %s found %d raw notices", sc.Name(), len(notices))

			for _, notice := range notices {
				mu.Lock()
				result.TotalScanned++
				mu.Unlock()

				hash := scrapers.GenerateNoticeHash(notice.SourcePortal, notice.NoticeTitle, notice.NoticeURL)
				exists, err := s.jobService.ExistsByHash(hash)
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
				details, err := s.extractor.Extract(notice)
				if err != nil {
					log.Printf("[Extractor Error] Failed for %s: %v", notice.NoticeTitle, err)
					mu.Lock()
					result.Errors++
					mu.Unlock()
					continue
				}

				j := &job.Job{
					Hash:         hash,
					SourcePortal: notice.SourcePortal,
					Status:       job.StatusDraft,
					RawNotice:    notice,
					Details:      *details,
					ReviewNotes:  "Auto-scraped with enriched metadata. Awaiting review.",
					CreatedAt:    time.Now(),
					UpdatedAt:    time.Now(),
				}

				if err := s.jobService.SaveDraft(j); err != nil {
					log.Printf("[DB Error] Failed to save draft: %v", err)
					mu.Lock()
					result.Errors++
					mu.Unlock()
				} else {
					log.Printf("[Draft Created] ID: %d | [%s] %s (Vacancies: %d, Last Date: %s)",
						j.ID, j.Details.Category, j.Details.Title, j.Details.TotalVacancies, j.Details.ApplicationEnd)

					// Auto-publish verified active notices with legitimate official document/portal URLs
					shouldPublish := j.Details.OfficialNoticeURL != "" && len(j.Details.Title) >= 6

					if shouldPublish {
						if _, pubErr := s.jobService.PublishJob(j.ID); pubErr == nil {
							log.Printf("[Notice Published] ID: %d | [%s] %s", j.ID, j.Details.NoticeType, j.Details.Title)
						}
					}

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

func (s *service) RunScraper(name string) PipelineResult {
	var result PipelineResult
	var target scrapers.PortalScraper

	for _, sc := range s.scrapers {
		if strings.EqualFold(sc.Name(), name) {
			target = sc
			break
		}
	}

	if target == nil {
		log.Printf("[Scraper] No scraper registered with name '%s'", name)
		return result
	}

	log.Printf("[Scraper] Running single scraper batch: %s (%s)...", target.Name(), target.BaseURL())
	notices, err := target.FetchNotices()
	if err != nil {
		log.Printf("[Scraper Error] %s: %v", target.Name(), err)
		result.Errors++
		return result
	}

	log.Printf("[Scraper] %s fetched %d raw notices", target.Name(), len(notices))

	for _, notice := range notices {
		result.TotalScanned++

		hash := scrapers.GenerateNoticeHash(notice.SourcePortal, notice.NoticeTitle, notice.NoticeURL)
		exists, err := s.jobService.ExistsByHash(hash)
		if err != nil {
			log.Printf("[DB Error] Exists check failed: %v", err)
			continue
		}

		if exists {
			result.Skipped++
			continue
		}

		// Extract structured details
		details, err := s.extractor.Extract(notice)
		if err != nil {
			log.Printf("[Extractor Error] Failed for %s: %v", notice.NoticeTitle, err)
			result.Errors++
			continue
		}

		j := &job.Job{
			Hash:         hash,
			SourcePortal: notice.SourcePortal,
			Status:       job.StatusDraft,
			RawNotice:    notice,
			Details:      *details,
			ReviewNotes:  "Auto-scraped with enriched metadata.",
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
		}

		if err := s.jobService.SaveDraft(j); err != nil {
			log.Printf("[DB Error] Failed to save draft: %v", err)
			result.Errors++
		} else {
			log.Printf("[Draft Created] ID: %d | [%s] %s (Vacancies: %d, Last Date: %s)",
				j.ID, j.Details.Category, j.Details.Title, j.Details.TotalVacancies, j.Details.ApplicationEnd)

			// Auto-publish verified active notices with legitimate official document/portal URLs
			shouldPublish := j.Details.OfficialNoticeURL != "" && len(j.Details.Title) >= 6

			if shouldPublish {
				if _, pubErr := s.jobService.PublishJob(j.ID); pubErr == nil {
					log.Printf("[Notice Published] ID: %d | [%s] %s", j.ID, j.Details.NoticeType, j.Details.Title)
				}
			}

			result.NewDrafts++
		}
	}

	log.Printf("[Scraper Batch Completed] %s: Scanned=%d, New=%d, Skipped=%d, Errors=%d",
		target.Name(), result.TotalScanned, result.NewDrafts, result.Skipped, result.Errors)
	return result
}

func (s *service) ReExtractDrafts() (int, int) {
	jobs, err := s.jobService.GetJobsForReExtraction()
	if err != nil {
		log.Printf("[ReExtract Error] Failed to fetch drafts: %v", err)
		return 0, 0
	}

	updatedCount := 0
	errorCount := 0

	for _, j := range jobs {
		details, err := s.extractor.Extract(j.RawNotice)
		if err != nil {
			errorCount++
			continue
		}

		j.Details = *details
		j.ReviewNotes = "Upgraded with enhanced heuristic data extractor"
		if err := s.jobService.UpdateJob(&j); err != nil {
			errorCount++
		} else {
			updatedCount++
		}
	}

	log.Printf("[ReExtract] Finished upgrading existing drafts: %d updated, %d errors", updatedCount, errorCount)
	return updatedCount, errorCount
}
