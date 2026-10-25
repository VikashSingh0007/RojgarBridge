package scrapers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"rozgarhub-backend/internal/models"
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
	CreatedAt   string          `json:"createdAt"`
	Attachments []sscAttachment `json:"attachments"`
}

type sscResponse struct {
	StatusCode    string          `json:"statusCode"`
	StatusMessage string          `json:"statusMessage"`
	Data          []sscNoticeItem `json:"data"`
}

func (s *SSCScraper) FetchNotices() ([]models.RawNotice, error) {
	apiURL := "https://ssc.gov.in/api/general-website/portal/notice-boards?page=1&limit=25&order=DESC&isAttachment=true&attributes=id,headline,startDate,endDate,language"

	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "application/json, text/plain, */*")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("ssc api request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ssc api returned status: %d", resp.StatusCode)
	}

	var sscResp sscResponse
	if err := json.NewDecoder(resp.Body).Decode(&sscResp); err != nil {
		return nil, fmt.Errorf("failed to decode ssc response: %w", err)
	}

	var notices []models.RawNotice

	for _, item := range sscResp.Data {
		// Prefer English notices for clean parsing
		if item.Language != "english" && item.Language != "" {
			continue
		}

		headline := strings.TrimSpace(item.Headline)
		if headline == "" {
			continue
		}

		// Construct direct official PDF URL
		pdfURL := "https://ssc.gov.in"
		if len(item.Attachments) > 0 && item.Attachments[0].Path != "" {
			cleanPath := strings.ReplaceAll(item.Attachments[0].Path, "\\", "/")
			pdfURL = fmt.Sprintf("https://ssc.gov.in/api/attachment/%s", strings.TrimPrefix(cleanPath, "/"))
		}

		notices = append(notices, models.RawNotice{
			SourcePortal: "SSC",
			NoticeTitle:  headline,
			NoticeURL:    pdfURL,
			PublishedAt:  item.CreatedAt,
			FetchedAt:    time.Now(),
			ContentText:  fmt.Sprintf("Title: %s | Source: Staff Selection Commission (Official) | Official PDF: %s", headline, pdfURL),
		})
	}

	return notices, nil
}
