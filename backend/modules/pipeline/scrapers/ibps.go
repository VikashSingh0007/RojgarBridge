package scrapers

import (
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"rojgarbridge-backend/modules/job"
	"rojgarbridge-backend/utils"
)

type IBPSScraper struct {
	client *http.Client
}

func NewIBPSScraper() *IBPSScraper {
	return &IBPSScraper{
		client: utils.NewSecureHTTPClient(45 * time.Second),
	}
}

func (s *IBPSScraper) Name() string {
	return "IBPS"
}

func (s *IBPSScraper) BaseURL() string {
	return "https://www.ibps.in"
}

// parseIBPSDate parses dates commonly found on IBPS portal (e.g. "Posted on 29 Oct, 2026", "29 Oct 2026", "27 Aug 2026")
func parseIBPSDate(raw string) (string, time.Time) {
	raw = strings.TrimSpace(raw)
	raw = strings.TrimPrefix(raw, "Posted on ")
	raw = strings.TrimSpace(raw)

	// Clean trailing punctuation
	raw = strings.ReplaceAll(raw, ",", " ")
	raw = strings.Join(strings.Fields(raw), " ")

	layouts := []string{
		"02 Jan 2006", "2 Jan 2006",
		"02 January 2006", "2 January 2006",
		"02-01-2006", "02/01/2006",
		"2006-01-02",
	}

	for _, layout := range layouts {
		if t, err := time.Parse(layout, raw); err == nil {
			return t.Format("2006-01-02"), t
		}
	}

	if raw != "" {
		return raw, time.Time{}
	}
	return time.Now().Format("2006-01-02"), time.Now()
}

func ensureAbsoluteIBPSURL(href string) string {
	href = strings.TrimSpace(href)
	if href == "" {
		return ""
	}
	if strings.HasPrefix(href, "http://") || strings.HasPrefix(href, "https://") {
		return href
	}
	return "https://www.ibps.in/" + strings.TrimPrefix(href, "/")
}

func (s *IBPSScraper) fetchURLWithRetry(targetURL string) (*http.Response, error) {
	var resp *http.Response
	var err error
	for attempt := 0; attempt < 3; attempt++ {
		req, reqErr := http.NewRequest("GET", targetURL, nil)
		if reqErr != nil {
			return nil, reqErr
		}
		req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36")
		req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")

		resp, err = s.client.Do(req)
		if err == nil && resp.StatusCode == http.StatusOK {
			return resp, nil
		}
		if resp != nil {
			resp.Body.Close()
		}
		time.Sleep(500 * time.Millisecond)
	}
	return resp, err
}

func (s *IBPSScraper) FetchNotices() ([]job.RawNotice, error) {
	var notices []job.RawNotice
	seen := make(map[string]bool)

	maxAge := 180 * 24 * time.Hour
	now := time.Now()

	// 1. Fetch Homepage Custom-Cards (Latest primary highlights)
	s.fetchHomepageCards(&notices, seen, maxAge, now)

	// 2. Fetch Careers Page Tenders (Direct recruitments and online applications)
	s.fetchCareersTenders(&notices, seen, maxAge, now)

	// 3. Fetch Active CRP Updates (PO/MT, Clerical, Specialist, and RRB processes)
	s.fetchCRPProcessNotices(&notices, seen, maxAge, now)

	log.Printf("[IBPS Scraper] Discovered %d live notices from official portal", len(notices))
	return notices, nil
}

func (s *IBPSScraper) fetchHomepageCards(notices *[]job.RawNotice, seen map[string]bool, maxAge time.Duration, now time.Time) {
	resp, err := s.fetchURLWithRetry("https://www.ibps.in/")
	if err != nil {
		log.Printf("[IBPS Scraper] Failed to fetch homepage cards: %v", err)
		return
	}
	defer resp.Body.Close()

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return
	}

	doc.Find(".custom-card").Each(func(i int, card *goquery.Selection) {
		heading := strings.TrimSpace(card.Find(".heading h3").Text())
		if heading == "" {
			heading = strings.TrimSpace(card.Find(".heading a").Text())
		}
		subHeading := strings.TrimSpace(card.Find(".sub-heading, p.sub-heading").Text())
		if subHeading == "" {
			subHeading = strings.TrimSpace(card.Find("p").Text())
		}
		dateText := strings.TrimSpace(card.Find(".crpdate").Text())

		if subHeading == "" {
			subHeading = heading
		}
		if subHeading == "" {
			return
		}

		title := subHeading
		lowerHeading := strings.ToLower(heading)
		if heading != "" && !strings.Contains(subHeading, heading) && !strings.Contains(lowerHeading, "advertisement") && !strings.Contains(lowerHeading, "career") {
			title = fmt.Sprintf("[%s] %s", heading, subHeading)
		}

		// Find best link: prefer PDF or application link from sub-heading or heading
		var targetURL string
		card.Find("a[href]").Each(func(j int, a *goquery.Selection) {
			href, _ := a.Attr("href")
			href = strings.TrimSpace(href)
			if href != "" {
				if strings.HasSuffix(strings.ToLower(href), ".pdf") {
					targetURL = href
				} else if targetURL == "" {
					targetURL = href
				}
			}
		})

		if targetURL == "" {
			targetURL = "https://www.ibps.in"
		}
		targetURL = ensureAbsoluteIBPSURL(targetURL)

		pubDate, t := parseIBPSDate(dateText)
		if !t.IsZero() && now.Sub(t) > maxAge {
			return // Skip outside dynamic lifecycle window
		}

		if seen[title] || seen[targetURL] {
			return
		}
		seen[title] = true
		seen[targetURL] = true

		contentText := fmt.Sprintf("Title: %s | Source: Institute of Banking Personnel Selection (Official Portal: ibps.in) | Published Date: %s | Official Link: %s", title, pubDate, targetURL)

		*notices = append(*notices, job.RawNotice{
			SourcePortal: "IBPS",
			NoticeTitle:  title,
			NoticeURL:    targetURL,
			PublishedAt:  pubDate,
			FetchedAt:    now,
			ContentText:  contentText,
		})
	})
}

func (s *IBPSScraper) fetchCareersTenders(notices *[]job.RawNotice, seen map[string]bool, maxAge time.Duration, now time.Time) {
	resp, err := s.fetchURLWithRetry("https://www.ibps.in/index.php/careers/")
	if err != nil {
		log.Printf("[IBPS Scraper] Failed to fetch careers tenders: %v", err)
		return
	}
	defer resp.Body.Close()

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return
	}

	doc.Find(".tender").Each(func(i int, sel *goquery.Selection) {
		title := strings.TrimSpace(sel.Find(".first-heading").Text())
		dateStr := strings.TrimSpace(sel.Find(".second-heading").Text())
		link, _ := sel.Find(".third-heading a[href]").Attr("href")
		link = ensureAbsoluteIBPSURL(link)

		if title == "" {
			return
		}
		if link == "" {
			link = "https://www.ibps.in/index.php/careers/"
		}

		pubDate, t := parseIBPSDate(dateStr)
		if !t.IsZero() && now.Sub(t) > maxAge {
			return
		}

		if seen[title] || seen[link] {
			return
		}
		seen[title] = true
		seen[link] = true

		contentText := fmt.Sprintf("Title: %s | Source: Institute of Banking Personnel Selection (IBPS Careers) | Published Date: %s | Official Link: %s", title, pubDate, link)

		*notices = append(*notices, job.RawNotice{
			SourcePortal: "IBPS",
			NoticeTitle:  title,
			NoticeURL:    link,
			PublishedAt:  pubDate,
			FetchedAt:    now,
			ContentText:  contentText,
		})
	})
}

func (s *IBPSScraper) fetchCRPProcessNotices(notices *[]job.RawNotice, seen map[string]bool, maxAge time.Duration, now time.Time) {
	categoryPages := []string{
		"https://www.ibps.in/index.php/management-trainees/",
		"https://www.ibps.in/index.php/clerical-cadre/",
		"https://www.ibps.in/index.php/specialist-officers/",
		"https://www.ibps.in/index.php/regional-rural-bank/",
	}

	for _, catURL := range categoryPages {
		latestProcessURL := s.getLatestProcessURL(catURL)
		if latestProcessURL == "" {
			continue
		}

		s.fetchProcessDetailList(latestProcessURL, notices, seen, maxAge, now)
	}
}

func (s *IBPSScraper) getLatestProcessURL(catURL string) string {
	resp, err := s.fetchURLWithRetry(catURL)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return ""
	}

	// First icon-box link represents the latest active CRP edition
	var latestURL string
	doc.Find(".elementor-icon-box-title a[href]").Each(func(i int, a *goquery.Selection) {
		if latestURL != "" {
			return
		}
		href, exists := a.Attr("href")
		if exists && href != "" {
			latestURL = ensureAbsoluteIBPSURL(href)
		}
	})

	return latestURL
}

func (s *IBPSScraper) fetchProcessDetailList(processURL string, notices *[]job.RawNotice, seen map[string]bool, maxAge time.Duration, now time.Time) {
	resp, err := s.fetchURLWithRetry(processURL)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return
	}

	doc.Find(".detail-list").Each(func(i int, sel *goquery.Selection) {
		dateStr := strings.TrimSpace(sel.Find(".detail-first-heading").Text())
		title := strings.TrimSpace(sel.Find(".detail-second-heading").Text())
		if title == "" {
			return
		}

		link, _ := sel.Closest("a").Attr("href")
		if link == "" {
			link, _ = sel.ParentsFiltered("a").Attr("href")
		}
		link = ensureAbsoluteIBPSURL(link)
		if link == "" {
			link = processURL
		}

		pubDate, t := parseIBPSDate(dateStr)
		if !t.IsZero() && now.Sub(t) > maxAge {
			return
		}

		if seen[title] || seen[link] {
			return
		}
		seen[title] = true
		seen[link] = true

		contentText := fmt.Sprintf("Title: %s | Source: Institute of Banking Personnel Selection (IBPS CRP) | Published Date: %s | Official Document: %s", title, pubDate, link)

		*notices = append(*notices, job.RawNotice{
			SourcePortal: "IBPS",
			NoticeTitle:  title,
			NoticeURL:    link,
			PublishedAt:  pubDate,
			FetchedAt:    now,
			ContentText:  contentText,
		})
	})
}

