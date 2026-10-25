package scrapers

import (
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"rozgarhub-backend/internal/models"
)

type RSSItem struct {
	Title       string `xml:"title"`
	Link        string `xml:"link"`
	Description string `xml:"description"`
	PubDate     string `xml:"pubDate"`
}

type RSSFeed struct {
	Channel struct {
		Title string    `xml:"title"`
		Items []RSSItem `xml:"item"`
	} `xml:"channel"`
}

type RSSScraper struct {
	name    string
	feedURL string
	client  *http.Client
}

func NewRSSScraper(name, feedURL string) *RSSScraper {
	return &RSSScraper{
		name:    name,
		feedURL: feedURL,
		client: &http.Client{
			Timeout: 15 * time.Second,
		},
	}
}

func (s *RSSScraper) Name() string {
	return s.name
}

func (s *RSSScraper) BaseURL() string {
	return s.feedURL
}

func (s *RSSScraper) FetchNotices() ([]models.RawNotice, error) {
	req, err := http.NewRequest("GET", s.feedURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "RozgarHub-Bot/1.0 (+https://rozgarhub.in)")

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("feed returned status: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var feed RSSFeed
	if err := xml.Unmarshal(body, &feed); err != nil {
		return nil, fmt.Errorf("failed to parse RSS XML: %w", err)
	}

	var notices []models.RawNotice
	for _, item := range feed.Channel.Items {
		title := strings.TrimSpace(item.Title)
		lowerTitle := strings.ToLower(title)

		// Filter for job-related keywords
		if strings.Contains(lowerTitle, "recruitment") ||
			strings.Contains(lowerTitle, "vacancy") ||
			strings.Contains(lowerTitle, "exam") ||
			strings.Contains(lowerTitle, "bharti") ||
			strings.Contains(lowerTitle, "apply") ||
			strings.Contains(lowerTitle, "officer") ||
			strings.Contains(lowerTitle, "assistant") ||
			strings.Contains(lowerTitle, "inspector") {

			notices = append(notices, models.RawNotice{
				SourcePortal: s.name,
				NoticeTitle:  title,
				NoticeURL:    strings.TrimSpace(item.Link),
				PublishedAt:  item.PubDate,
				FetchedAt:    time.Now(),
				ContentText:  item.Description,
			})
		}
	}

	return notices, nil
}
