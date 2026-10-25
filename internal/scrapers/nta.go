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

type NTAScraper struct {
	client *http.Client
}

func NewNTAScraper() *NTAScraper {
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	return &NTAScraper{
		client: &http.Client{
			Transport: tr,
			Timeout:   30 * time.Second,
		},
	}
}

func (s *NTAScraper) Name() string {
	return "NTA"
}

func (s *NTAScraper) BaseURL() string {
	return "https://nta.ac.in"
}

func (s *NTAScraper) FetchNotices() ([]models.RawNotice, error) {
	req, err := http.NewRequest("GET", "https://nta.ac.in/", nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("nta fetch failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("nta status: %d", resp.StatusCode)
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to parse nta html: %w", err)
	}

	var notices []models.RawNotice
	seen := make(map[string]bool)

	// Look for notice links and pdfs
	doc.Find("a[href*='.pdf'], a[href*='Notice']").Each(func(i int, sel *goquery.Selection) {
		title := strings.TrimSpace(sel.Text())
		href, exists := sel.Attr("href")
		if !exists || title == "" {
			return
		}

		// If title is just "Read More", get parent or sibling text
		if strings.EqualFold(title, "Read More") || strings.EqualFold(title, "Read More ") {
			parent := sel.Parent()
			siblingText := strings.TrimSpace(parent.Find("content, p, span").Text())
			if siblingText != "" {
				title = siblingText
			} else {
				return
			}
		}

		if len(title) < 5 {
			return
		}

		if !strings.HasPrefix(href, "http://") && !strings.HasPrefix(href, "https://") {
			href = "https://nta.ac.in" + "/" + strings.TrimPrefix(href, "/")
		}

		if seen[href] || seen[title] {
			return
		}
		seen[href] = true
		seen[title] = true

		notices = append(notices, models.RawNotice{
			SourcePortal: "NTA",
			NoticeTitle:  title,
			NoticeURL:    href,
			PublishedAt:  time.Now().Format("2006-01-02"),
			FetchedAt:    time.Now(),
			ContentText:  fmt.Sprintf("Title: %s | Source: National Testing Agency (Official Portal: nta.ac.in) | Official Document: %s", title, href),
		})
	})

	return notices, nil
}
