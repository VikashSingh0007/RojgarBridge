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

type ISROScraper struct {
	client *http.Client
}

func NewISROScraper() *ISROScraper {
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	return &ISROScraper{
		client: &http.Client{
			Transport: tr,
			Timeout:   30 * time.Second,
		},
	}
}

func (s *ISROScraper) Name() string {
	return "ISRO"
}

func (s *ISROScraper) BaseURL() string {
	return "https://www.isro.gov.in"
}

func (s *ISROScraper) FetchNotices() ([]models.RawNotice, error) {
	req, err := http.NewRequest("GET", "https://www.isro.gov.in/Careers.html", nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("isro fetch failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("isro status: %d", resp.StatusCode)
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to parse isro html: %w", err)
	}

	var notices []models.RawNotice
	seen := make(map[string]bool)

	doc.Find("table.content-table td.news a, table td.news a").Each(func(i int, sel *goquery.Selection) {
		title := strings.TrimSpace(sel.AttrOr("title", ""))
		if title == "" {
			title = strings.TrimSpace(sel.Text())
		}
		title = strings.TrimSuffix(title, "Read More")
		title = strings.TrimSpace(title)

		href, exists := sel.Attr("href")
		if !exists || title == "" || len(title) < 10 {
			return
		}

		if !strings.HasPrefix(href, "http://") && !strings.HasPrefix(href, "https://") {
			href = "https://www.isro.gov.in/" + strings.TrimPrefix(href, "/")
		}

		if seen[title] || seen[href] {
			return
		}
		seen[title] = true
		seen[href] = true

		notices = append(notices, models.RawNotice{
			SourcePortal: "ISRO",
			NoticeTitle:  title,
			NoticeURL:    href,
			PublishedAt:  time.Now().Format("2006-01-02"),
			FetchedAt:    time.Now(),
			ContentText:  fmt.Sprintf("Title: %s | Source: Indian Space Research Organisation (Official Portal: isro.gov.in) | Category: Engineering & PSU | Link: %s", title, href),
		})
	})

	return notices, nil
}
