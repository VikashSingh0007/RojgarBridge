package pipeline

import (
	"context"
	"log"
	"sync"
	"time"
)

type Processor struct {
	service  Service
	interval time.Duration
	stopChan chan struct{}
	wg       sync.WaitGroup
	mu       sync.Mutex
	running  bool
}

func NewProcessor(svc Service, interval time.Duration) *Processor {
	if interval <= 0 {
		interval = 6 * time.Hour
	}
	return &Processor{
		service:  svc,
		interval: interval,
		stopChan: make(chan struct{}),
	}
}

func (p *Processor) Start(ctx context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.running {
		return nil
	}
	p.running = true
	p.stopChan = make(chan struct{})

	p.wg.Add(1)
	go func() {
		defer p.wg.Done()

		log.Println("[Pipeline Processor] Running initial scraping pass...")
		res := p.service.RunPipeline()
		log.Printf("[Pipeline Processor] Initial pass completed: Scanned: %d, New Drafts: %d, Skipped: %d, Errors: %d",
			res.TotalScanned, res.NewDrafts, res.Skipped, res.Errors)

		ticker := time.NewTicker(p.interval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				log.Println("[Pipeline Processor] Context canceled, shutting down...")
				return
			case <-p.stopChan:
				log.Println("[Pipeline Processor] Stop signal received, shutting down...")
				return
			case <-ticker.C:
				log.Println("[Pipeline Processor] Running periodic scraper pass...")
				passRes := p.service.RunPipeline()
				log.Printf("[Pipeline Processor] Pass completed: Scanned: %d, New Drafts: %d, Skipped: %d, Errors: %d",
					passRes.TotalScanned, passRes.NewDrafts, passRes.Skipped, passRes.Errors)
			}
		}
	}()

	return nil
}

func (p *Processor) Stop() error {
	p.mu.Lock()
	if !p.running {
		p.mu.Unlock()
		return nil
	}
	p.running = false
	close(p.stopChan)
	p.mu.Unlock()

	p.wg.Wait()
	log.Println("[Pipeline Processor] Processor stopped successfully.")
	return nil
}
