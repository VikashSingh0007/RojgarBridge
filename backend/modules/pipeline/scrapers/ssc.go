package scrapers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"rojgarbridge-backend/modules/job"
	"rojgarbridge-backend/utils"
)

type SSCScraper struct {
	client *http.Client
}

func NewSSCScraper() *SSCScraper {
	return &SSCScraper{
		client: utils.NewSecureHTTPClient(45 * time.Second),
	}
}

func (s *SSCScraper) Name() string {
	return "SSC"
}

func (s *SSCScraper) BaseURL() string {
	return "https://ssc.gov.in"
}

type sscAttachment struct {
	ID       string `json:"id"`
	FileName string `json:"fileName"`
	Type     string `json:"type"`
	Size     int64  `json:"size"`
	Path     string `json:"path"`
}

type sscNoticeItem struct {
	ID          string          `json:"id"`
	Headline    string          `json:"headline"`
	ExamID      string          `json:"examId"`
	ContentType string          `json:"contentType"`
	Language    string          `json:"language"`
	StartDate   string          `json:"startDate"`
	EndDate     string          `json:"endDate"`
	CreatedAt   string          `json:"createdAt"`
	Attachments []sscAttachment `json:"attachments"`
}

type sscResponse struct {
	StatusCode    string          `json:"statusCode"`
	StatusMessage string          `json:"statusMessage"`
	Data          []sscNoticeItem `json:"data"`
}

func (s *SSCScraper) FetchNotices() ([]job.RawNotice, error) {
	// Query official SSC live portal notices endpoint
	apiURL := "https://ssc.gov.in/api/general-website/portal/notice-boards?page=1&limit=50&key=createdAt&order=DESC&isPaginationRequired=false&isAttachment=true&language=english&attributes=id,headline,examId,contentType,startDate,endDate,language,createdAt"

	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create SSC request: %w", err)
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "application/json, text/plain, */*")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch SSC notices: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("SSC API returned HTTP %d", resp.StatusCode)
	}

	var sscResp sscResponse
	if err := json.NewDecoder(resp.Body).Decode(&sscResp); err != nil {
		return nil, fmt.Errorf("failed to decode SSC JSON response: %w", err)
	}

	var notices []job.RawNotice
	seen := make(map[string]bool)

	// Lifecycle is data-driven: filter relative to time.Now() (default 180 days window)
	maxAge := 180 * 24 * time.Hour
	now := time.Now()

	for _, item := range sscResp.Data {
		// Prefer English notices when language tag is specified
		if item.Language != "english" && item.Language != "" {
			continue
		}

		headline := strings.TrimSpace(item.Headline)
		if headline == "" {
			continue
		}

		// Parse dynamic creation date from source
		pubDate := item.CreatedAt
		if t, parseErr := time.Parse(time.RFC3339, item.CreatedAt); parseErr == nil {
			if now.Sub(t) > maxAge {
				continue // Skip notices outside dynamic lifecycle window
			}
			pubDate = t.Format("2006-01-02")
		} else if len(pubDate) > 10 {
			pubDate = pubDate[:10]
		}

		// Resolve official attachment PDF URL from live response
		pdfURL := "https://ssc.gov.in"
		if len(item.Attachments) > 0 && item.Attachments[0].Path != "" {
			cleanPath := strings.ReplaceAll(item.Attachments[0].Path, "\\", "/")
			pdfURL = fmt.Sprintf("https://ssc.gov.in/api/attachment/%s", strings.TrimPrefix(cleanPath, "/"))
		}

		// Deduplicate by live notice ID and attachment URL
		dedupKey := item.ID
		if dedupKey == "" {
			dedupKey = pdfURL
		}
		if seen[dedupKey] || seen[pdfURL] {
			continue
		}
		seen[dedupKey] = true
		seen[pdfURL] = true

		// Use the source data truthfully as discovered from the portal
		notices = append(notices, job.RawNotice{
			SourcePortal: "SSC",
			NoticeTitle:  headline,
			NoticeURL:    pdfURL,
			PublishedAt:  pubDate,
			FetchedAt:    now,
			ContentText:  headline,
		})
	}

	log.Printf("[SSC Scraper] Discovered %d live notices from official portal", len(notices))
	return notices, nil
}
