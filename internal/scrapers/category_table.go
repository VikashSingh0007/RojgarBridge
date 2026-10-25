package scrapers

import (
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"rojgarbridge-backend/internal/models"
)

// CategoryTableScraper scrapes structured HTML tables from job categories
type CategoryTableScraper struct {
	name            string
	url             string
	defaultCategory string
	client          *http.Client
	maxItems        int
}

func NewCategoryTableScraper(name, url, defaultCategory string, maxItems int) *CategoryTableScraper {
	return &CategoryTableScraper{
		name:            name,
		url:             url,
		defaultCategory: defaultCategory,
		client: &http.Client{
			Timeout: 45 * time.Second,
		},
		maxItems: maxItems, // 0 means no limit (scrape all)
	}
}

func (s *CategoryTableScraper) Name() string {
	return s.name
}

func (s *CategoryTableScraper) BaseURL() string {
	return s.url
}

func (s *CategoryTableScraper) FetchNotices() ([]models.RawNotice, error) {
	req, err := http.NewRequest("GET", s.url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request for %s: %w", s.url, err)
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9,hi;q=0.8")

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch %s: %w", s.url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("status %d from %s", resp.StatusCode, s.url)
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to parse html from %s: %w", s.url, err)
	}

	var notices []models.RawNotice
	seenHashes := make(map[string]bool)

	// 1. Check for standard recruitment table rows: tr.lattrbord
	doc.Find("table.lattbl tr.lattrbord, tr.lattrbord").Each(func(i int, tr *goquery.Selection) {
		if s.maxItems > 0 && len(notices) >= s.maxItems {
			return
		}

		postDate := strings.TrimSpace(tr.Find("td.latcpb").Text())
		board := strings.TrimSpace(tr.Find("td.latcr").Text())
		postName := strings.TrimSpace(tr.Find("td.latceb").Text())
		qualification := strings.TrimSpace(tr.Find("td.latcqb").Text())
		advtNo := strings.TrimSpace(tr.Find("td.latcab").Text())
		lastDate := strings.TrimSpace(tr.Find("td.latclb").Text())

		linkSel := tr.Find("td.latcmb a").First()
		if linkSel.Length() == 0 {
			linkSel = tr.Find("a").First()
		}
		href, _ := linkSel.Attr("href")
		href = strings.TrimSpace(href)

		if postName == "" || board == "" {
			return
		}

		// Clean placeholders
		if advtNo == "–" || advtNo == "-" || advtNo == "&#8211;" {
			advtNo = ""
		}

		// Clean HTML entities from postName
		postName = cleanHTMLText(postName)

		title := fmt.Sprintf("[%s] %s", board, postName)
		if strings.HasPrefix(strings.ToLower(postName), strings.ToLower(board)) {
			title = postName
		}

		// Fallback URL if missing
		if href == "" {
			href = s.url
		}

		// Build rich structured content text for downstream extractor
		contentText := fmt.Sprintf(
			"Board: %s | Post: %s | Qualification: %s | Advt: %s | Last Date: %s | Category: %s | Source URL: %s",
			board, postName, qualification, advtNo, lastDate, s.defaultCategory, href,
		)

		uniqueKey := fmt.Sprintf("%s|%s|%s", board, postName, href)
		if seenHashes[uniqueKey] {
			return
		}
		seenHashes[uniqueKey] = true

		notices = append(notices, models.RawNotice{
			SourcePortal: s.name,
			NoticeTitle:  title,
			NoticeURL:    href,
			AdvtNumber:   advtNo,
			PublishedAt:  postDate,
			FetchedAt:    time.Now(),
			ContentText:  contentText,
		})
	})

	// 2. If no lattrbord found (e.g. Admit Card or Exam Result 4-column tables: Date, Board, Name, Link)
	if len(notices) == 0 {
		doc.Find("table tr").Each(func(i int, tr *goquery.Selection) {
			if s.maxItems > 0 && len(notices) >= s.maxItems {
				return
			}

			tds := tr.Find("td")
			if tds.Length() < 4 {
				return
			}

			col0 := strings.TrimSpace(tds.Eq(0).Text())
			col1 := strings.TrimSpace(tds.Eq(1).Text())
			col2 := strings.TrimSpace(tds.Eq(2).Text())

			// Skip header row
			if strings.EqualFold(col0, "post date") || strings.EqualFold(col1, "recruitment board") {
				return
			}

			linkSel := tds.Eq(3).Find("a").First()
			href, _ := linkSel.Attr("href")
			href = strings.TrimSpace(href)

			if col1 == "" || col2 == "" {
				return
			}

			title := fmt.Sprintf("[%s] %s (%s)", col1, col2, s.name)
			contentText := fmt.Sprintf(
				"Board: %s | Post/Exam: %s | Date: %s | Section: %s | Link: %s",
				col1, col2, col0, s.name, href,
			)

			uniqueKey := fmt.Sprintf("%s|%s|%s", col1, col2, s.name)
			if seenHashes[uniqueKey] {
				return
			}
			seenHashes[uniqueKey] = true

			notices = append(notices, models.RawNotice{
				SourcePortal: s.name,
				NoticeTitle:  title,
				NoticeURL:    href,
				AdvtNumber:   "",
				PublishedAt:  col0,
				FetchedAt:    time.Now(),
				ContentText:  contentText,
			})
		})
	}

	return notices, nil
}

func cleanHTMLText(s string) string {
	s = strings.ReplaceAll(s, "&#8211;", "–")
	s = strings.ReplaceAll(s, "&#8217;", "'")
	s = strings.ReplaceAll(s, "&amp;", "&")
	s = strings.ReplaceAll(s, "&nbsp;", " ")
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.ReplaceAll(s, "\r", " ")
	s = strings.ReplaceAll(s, "\t", " ")
	spaceRegex := regexp.MustCompile(`\s+`)
	s = spaceRegex.ReplaceAllString(s, " ")
	return strings.TrimSpace(s)
}

// Helper to extract vacancies from string
func extractVacanciesCount(text string) int {
	re := regexp.MustCompile(`(?i)(?:–|-|:)?\s*(\d{1,6})\s*(?:posts?|vacanc(?:y|ies)|seats?)`)
	match := re.FindStringSubmatch(text)
	if len(match) > 1 {
		if v, err := strconv.Atoi(match[1]); err == nil {
			return v
		}
	}
	return 0
}
