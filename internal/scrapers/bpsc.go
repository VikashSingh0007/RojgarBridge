package scrapers

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"rojgarbridge-backend/internal/models"
)

type BPSCScraper struct {
	client *http.Client
}

func NewBPSCScraper() *BPSCScraper {
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	return &BPSCScraper{
		client: &http.Client{
			Transport: tr,
			Timeout:   30 * time.Second,
		},
	}
}

func (s *BPSCScraper) Name() string {
	return "BPSC"
}

func (s *BPSCScraper) BaseURL() string {
	return "https://bpsc.bihar.gov.in"
}

func (s *BPSCScraper) FetchNotices() ([]models.RawNotice, error) {
	req, err := http.NewRequest("GET", "https://bpsc.bihar.gov.in/", nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("bpsc fetch failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("bpsc status: %d", resp.StatusCode)
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to parse bpsc html: %w", err)
	}

	var notices []models.RawNotice
	seen := make(map[string]bool)

	doc.Find("a[href*='.pdf']").Each(func(i int, sel *goquery.Selection) {
		href, exists := sel.Attr("href")
		if !exists {
			return
		}

		title := strings.TrimSpace(sel.Text())
		if title == "" || len(title) < 8 {
			return
		}

		if !strings.HasPrefix(href, "http://") && !strings.HasPrefix(href, "https://") {
			href = "https://bpsc.bihar.gov.in" + "/" + strings.TrimPrefix(href, "/")
		}

		if seen[href] || seen[title] {
			return
		}
		seen[href] = true
		seen[title] = true

		notices = append(notices, models.RawNotice{
			SourcePortal: "BPSC",
			NoticeTitle:  title,
			NoticeURL:    href,
			PublishedAt:  time.Now().Format("2006-01-02"),
			FetchedAt:    time.Now(),
			ContentText:  fmt.Sprintf("Title: %s | Source: Bihar Public Service Commission (Official Portal: bpsc.bihar.gov.in) | Official Document: %s", title, href),
		})
	})

	return notices, nil
}
