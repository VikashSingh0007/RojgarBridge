package scrapers

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"rojgarbridge-backend/internal/models"
)

type UPSCScraper struct {
	client *http.Client
}

func NewUPSCScraper() *UPSCScraper {
	return &UPSCScraper{
		client: &http.Client{
			Timeout: 45 * time.Second,
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
	endpoints := []struct {
		url    string
		source string
	}{
		{url: "https://upsc.gov.in/whats-new", source: "UPSC What's New"},
		{url: "https://upsc.gov.in/examinations/active-exams", source: "UPSC Active Exams"},
	}

	var notices []models.RawNotice
	seen := make(map[string]bool)

	for _, ep := range endpoints {
		req, err := http.NewRequest("GET", ep.url, nil)
		if err != nil {
			continue
		}

		req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36")
		req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,*/*;q=0.8")
		req.Header.Set("Accept-Language", "en-US,en;q=0.9,hi;q=0.8")
		req.Header.Set("Connection", "keep-alive")

		resp, err := s.client.Do(req)
		if err != nil {
			continue
		}

		if resp.StatusCode != http.StatusOK {
			resp.Body.Close()
			continue
		}

		doc, err := goquery.NewDocumentFromReader(resp.Body)
		resp.Body.Close()
		if err != nil {
			continue
		}

		doc.Find("table tbody tr, .view-content .views-row, table.views-table tr").Each(func(i int, sel *goquery.Selection) {
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

			lowerTitle := strings.ToLower(title)
			isJobRelated := strings.Contains(lowerTitle, "recruitment") ||
				strings.Contains(lowerTitle, "examination") ||
				strings.Contains(lowerTitle, "vacancy") ||
				strings.Contains(lowerTitle, "post") ||
				strings.Contains(lowerTitle, "notification") ||
				strings.Contains(lowerTitle, "cgl") ||
				strings.Contains(lowerTitle, "nda") ||
				strings.Contains(lowerTitle, "cds") ||
				strings.Contains(lowerTitle, "civil services") ||
				strings.Contains(lowerTitle, "cse") ||
				strings.Contains(lowerTitle, "admit") ||
				strings.Contains(lowerTitle, "result") ||
				strings.Contains(lowerTitle, "interview") ||
				strings.Contains(lowerTitle, "advt")

			if !isJobRelated {
				return
			}

			if seen[href] {
				return
			}
			seen[href] = true

			dateText := strings.TrimSpace(sel.Find(".views-field-created, td:last-child").Text())

			notices = append(notices, models.RawNotice{
				SourcePortal: "UPSC",
				NoticeTitle:  title,
				NoticeURL:    href,
				PublishedAt:  dateText,
				FetchedAt:    time.Now(),
				ContentText:  fmt.Sprintf("Title: %s | Source: %s | Link: %s", title, ep.source, href),
			})
		})
	}

	return notices, nil
}
