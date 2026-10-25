package scrapers

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"rozgarhub-backend/internal/models"
)

type UPSCScraper struct {
	client *http.Client
}

func NewUPSCScraper() *UPSCScraper {
	return &UPSCScraper{
		client: &http.Client{
			Timeout: 20 * time.Second,
		},
	}
}

func (s *UPSCScraper) Name() string {
	return "UPSC"
}

func (s *UPSCScraper) BaseURL() string {
	return "https://upsc.gov.in"
}

func (s *UPSCScraper) FetchNotices() ([]models.RawNotice, error) {
	url := "https://upsc.gov.in/whats-new"
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/122.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("upsc fetch failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("upsc returned non-200 status: %d", resp.StatusCode)
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to parse upsc html: %w", err)
	}

	var notices []models.RawNotice

	// UPSC "What's New" table parsing
	doc.Find("table tbody tr, .view-content .views-row").Each(func(i int, sel *goquery.Selection) {
		linkSel := sel.Find("a").First()
		title := strings.TrimSpace(linkSel.Text())
		href, exists := linkSel.Attr("href")

		if !exists || title == "" {
			return
		}

		// Ensure absolute URL
		if !strings.HasPrefix(href, "http://") && !strings.HasPrefix(href, "https://") {
			href = "https://upsc.gov.in/" + strings.TrimPrefix(href, "/")
		}

		// Filter for recruitment / exam / vacancy related keywords
		lowerTitle := strings.ToLower(title)
		isJobRelated := strings.Contains(lowerTitle, "recruitment") ||
			strings.Contains(lowerTitle, "examination") ||
			strings.Contains(lowerTitle, "vacancy") ||
			strings.Contains(lowerTitle, "post") ||
			strings.Contains(lowerTitle, "notification") ||
			strings.Contains(lowerTitle, "cgl") ||
			strings.Contains(lowerTitle, "nda") ||
			strings.Contains(lowerTitle, "cds") ||
			strings.Contains(lowerTitle, "civil services")

		if !isJobRelated {
			return
		}

		dateText := strings.TrimSpace(sel.Find(".views-field-created, td:last-child").Text())

		notices = append(notices, models.RawNotice{
			SourcePortal: "UPSC",
			NoticeTitle:  title,
			NoticeURL:    href,
			PublishedAt:  dateText,
			FetchedAt:    time.Now(),
			ContentText:  fmt.Sprintf("Title: %s | Source: UPSC What's New | Link: %s", title, href),
		})
	})

	return notices, nil
}
