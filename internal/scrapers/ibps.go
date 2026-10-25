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

type IBPSScraper struct {
	client *http.Client
}

func NewIBPSScraper() *IBPSScraper {
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	return &IBPSScraper{
		client: &http.Client{
			Transport: tr,
			Timeout:   30 * time.Second,
		},
	}
}

func (s *IBPSScraper) Name() string {
	return "IBPS"
}

func (s *IBPSScraper) BaseURL() string {
	return "https://www.ibps.in"
}

func (s *IBPSScraper) FetchNotices() ([]models.RawNotice, error) {
	req, err := http.NewRequest("GET", "https://www.ibps.in/", nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("ibps fetch failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ibps status: %d", resp.StatusCode)
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to parse ibps html: %w", err)
	}

	var notices []models.RawNotice
	seen := make(map[string]bool)

	// Parse IBPS custom-card announcement elements
	doc.Find(".custom-card").Each(func(i int, card *goquery.Selection) {
		heading := strings.TrimSpace(card.Find(".heading h3, .heading").Text())
		subHeading := strings.TrimSpace(card.Find(".sub-heading, p").Text())
		dateText := strings.TrimSpace(card.Find(".crpdate").Text())

		linkSel := card.Find("a[href]").First()
		href, _ := linkSel.Attr("href")
		href = strings.TrimSpace(href)

		if subHeading == "" {
			subHeading = heading
		}
		if subHeading == "" {
			return
		}

		title := subHeading
		if heading != "" && !strings.Contains(subHeading, heading) {
			title = fmt.Sprintf("[%s] %s", heading, subHeading)
		}

		if href == "" {
			href = "https://www.ibps.in"
		}

		if seen[title] || seen[href] {
			return
		}
		seen[title] = true
		seen[href] = true

		notices = append(notices, models.RawNotice{
			SourcePortal: "IBPS",
			NoticeTitle:  title,
			NoticeURL:    href,
			PublishedAt:  dateText,
			FetchedAt:    time.Now(),
			ContentText:  fmt.Sprintf("Title: %s | Source: Institute of Banking Personnel Selection (Official Portal: ibps.in) | Date: %s | Official Link: %s", title, dateText, href),
		})
	})

	return notices, nil
}
