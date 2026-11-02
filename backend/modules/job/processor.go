package job

import (
	"context"
	"log"
	"sync"
)

type Processor struct {
	service Service
	mu      sync.Mutex
	running bool
}

func NewProcessor(svc Service) *Processor {
	return &Processor{service: svc}
}

func (p *Processor) Start(ctx context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.running {
		return nil
	}
	p.running = true
	log.Println("[Job Processor] Started background processor.")
	return nil
}

func (p *Processor) Stop() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.running {
		return nil
	}
	p.running = false
	log.Println("[Job Processor] Stopped background processor.")
	return nil
}
