package scrapers

import (
	"crypto/sha256"
	"fmt"

	"rojgarbridge-backend/modules/job"
)

// PortalScraper interface implemented by each government portal scraper
type PortalScraper interface {
	Name() string
	BaseURL() string
	FetchNotices() ([]job.RawNotice, error)
}

// GenerateNoticeHash creates a unique fingerprint for a notice to prevent duplicates
func GenerateNoticeHash(portal, advtOrTitle, noticeURL string) string {
	data := fmt.Sprintf("%s|%s|%s", portal, advtOrTitle, noticeURL)
	h := sha256.New()
	h.Write([]byte(data))
	return fmt.Sprintf("%x", h.Sum(nil))
}
