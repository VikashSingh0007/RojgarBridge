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

type DSSSBScraper struct {
	client *http.Client
}

func NewDSSSBScraper() *DSSSBScraper {
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	return &DSSSBScraper{
		client: &http.Client{
			Transport: tr,
			Timeout:   30 * time.Second,
		},
	}
}

func (s *DSSSBScraper) Name() string {
	return "DSSSB"
}

func (s *DSSSBScraper) BaseURL() string {
	return "https://dsssb.delhi.gov.in"
}

func (s *DSSSBScraper) FetchNotices() ([]models.RawNotice, error) {
	req, err := http.NewRequest("GET", "https://dsssb.delhi.gov.in/notice-of-exam", nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("dsssb fetch failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("dsssb status: %d", resp.StatusCode)
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to parse dsssb html: %w", err)
	}

	var notices []models.RawNotice
	seen := make(map[string]bool)

	// DSSSB lists exam notices with title attribute or link text
	doc.Find("a[href*='.pdf']").Each(func(i int, sel *goquery.Selection) {
		href, exists := sel.Attr("href")
		if !exists {
			return
		}

		title := strings.TrimSpace(sel.AttrOr("title", ""))
		if title == "" {
			title = strings.TrimSpace(sel.Text())
		}

		if title == "" || strings.EqualFold(title, "Download") || len(title) < 8 {
			// Check parent table row for description
			rowText := strings.TrimSpace(sel.Closest("tr").Find("td:first-child").Text())
			if rowText != "" {
				title = rowText
			} else {
				return
			}
		}

		if !strings.HasPrefix(href, "http://") && !strings.HasPrefix(href, "https://") {
			href = "https://dsssb.delhi.gov.in" + "/" + strings.TrimPrefix(href, "/")
		}

		if seen[href] || seen[title] {
			return
		}
		seen[href] = true
		seen[title] = true

		notices = append(notices, models.RawNotice{
			SourcePortal: "DSSSB",
			NoticeTitle:  title,
			NoticeURL:    href,
			PublishedAt:  time.Now().Format("2006-01-02"),
			FetchedAt:    time.Now(),
			ContentText:  fmt.Sprintf("Title: %s | Source: Delhi Subordinate Services Selection Board (Official Portal: dsssb.delhi.gov.in) | Official Document: %s", title, href),
		})
	})

	return notices, nil
}
