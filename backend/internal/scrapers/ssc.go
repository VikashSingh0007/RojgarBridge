package scrapers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"rojgarbridge-backend/internal/models"
)

type SSCScraper struct {
	client *http.Client
}

func NewSSCScraper() *SSCScraper {
	return &SSCScraper{
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (s *SSCScraper) Name() string {
	return "SSC"
}

func (s *SSCScraper) BaseURL() string {
	return "https://ssc.gov.in"
}

// SSC API Response structures
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

func (s *SSCScraper) FetchNotices() ([]models.RawNotice, error) {
	var notices []models.RawNotice
	seen := make(map[string]bool)

	// Fetch up to 3 pages (up to 150 official notices from Staff Selection Commission)
	for page := 1; page <= 3; page++ {
		apiURL := fmt.Sprintf("https://ssc.gov.in/api/general-website/portal/notice-boards?page=%d&limit=50&order=DESC&isAttachment=true&attributes=id,headline,language", page)

		req, err := http.NewRequest("GET", apiURL, nil)
		if err != nil {
			continue
		}

		req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36")
		req.Header.Set("Accept", "application/json, text/plain, */*")
		req.Header.Set("Accept-Language", "en-US,en;q=0.9")

		resp, err := s.client.Do(req)
		if err != nil {
			continue
		}

		if resp.StatusCode != http.StatusOK {
			resp.Body.Close()
			continue
		}

		var sscResp sscResponse
		if err := json.NewDecoder(resp.Body).Decode(&sscResp); err != nil {
			resp.Body.Close()
			continue
		}
		resp.Body.Close()

		for _, item := range sscResp.Data {
			// Prefer English notices
			if item.Language != "english" && item.Language != "" {
				continue
			}

			headline := strings.TrimSpace(item.Headline)
			if headline == "" {
				continue
			}

			// Direct official SSC attachment PDF
			pdfURL := "https://ssc.gov.in"
			if len(item.Attachments) > 0 && item.Attachments[0].Path != "" {
				cleanPath := strings.ReplaceAll(item.Attachments[0].Path, "\\", "/")
				pdfURL = fmt.Sprintf("https://ssc.gov.in/api/attachment/%s", strings.TrimPrefix(cleanPath, "/"))
			}

			if seen[pdfURL] || seen[headline] {
				continue
			}
			seen[pdfURL] = true
			seen[headline] = true

			pubDate := item.CreatedAt
			if len(pubDate) > 10 {
				pubDate = pubDate[:10]
			}

			notices = append(notices, models.RawNotice{
				SourcePortal: "SSC",
				NoticeTitle:  headline,
				NoticeURL:    pdfURL,
				PublishedAt:  pubDate,
				FetchedAt:    time.Now(),
				ContentText:  fmt.Sprintf("Title: %s | Source: Staff Selection Commission (Official Portal: ssc.gov.in) | Official Document: %s", headline, pdfURL),
			})
		}
	}

	return notices, nil
}
