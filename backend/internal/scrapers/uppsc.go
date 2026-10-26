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

type UPPSCScraper struct {
	client *http.Client
}

func NewUPPSCScraper() *UPPSCScraper {
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	return &UPPSCScraper{
		client: &http.Client{
			Transport: tr,
			Timeout:   30 * time.Second,
		},
	}
}

func (s *UPPSCScraper) Name() string {
	return "UPPSC"
}

func (s *UPPSCScraper) BaseURL() string {
	return "https://uppsc.up.nic.in"
}

func (s *UPPSCScraper) FetchNotices() ([]models.RawNotice, error) {
	req, err := http.NewRequest("GET", "https://uppsc.up.nic.in/", nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("uppsc fetch failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("uppsc status: %d", resp.StatusCode)
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to parse uppsc html: %w", err)
	}

	var notices []models.RawNotice
	seen := make(map[string]bool)

	doc.Find("#ctl00_MainContent_RecruitmentDash a, marquee a").Each(func(i int, sel *goquery.Selection) {
		title := strings.TrimSpace(sel.AttrOr("title", ""))
		if title == "" {
			title = strings.TrimSpace(sel.Text())
		}
		href, exists := sel.Attr("href")
		if !exists || title == "" || len(title) < 10 {
			return
		}

		title = strings.TrimPrefix(title, "CLICK HERE TO APPLY FOR ")
		title = strings.TrimPrefix(title, "CLICK HERE FOR ")
		title = strings.TrimPrefix(title, "CLICK HERE TO DOWNLOAD ")
		title = strings.TrimSpace(title)

		if !strings.HasPrefix(href, "http://") && !strings.HasPrefix(href, "https://") {
			href = "https://uppsc.up.nic.in/" + strings.TrimPrefix(href, "/")
		}

		if seen[title] || seen[href] {
			return
		}
		seen[title] = true
		seen[href] = true

		notices = append(notices, models.RawNotice{
			SourcePortal: "UPPSC",
			NoticeTitle:  title,
			NoticeURL:    href,
			PublishedAt:  time.Now().Format("2006-01-02"),
			FetchedAt:    time.Now(),
			ContentText:  fmt.Sprintf("Title: %s | Source: Uttar Pradesh Public Service Commission (Official Portal: uppsc.up.nic.in) | State: Uttar Pradesh | Link: %s", title, href),
		})
	})

	return notices, nil
}
