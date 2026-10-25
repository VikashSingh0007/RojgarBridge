package scrapers

import (
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"rojgarbridge-backend/internal/models"
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
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "application/rss+xml, application/xml, text/xml, */*")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")

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
			strings.Contains(lowerTitle, "inspector") ||
			strings.Contains(lowerTitle, "posts") ||
			strings.Contains(lowerTitle, "walkin") {

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
