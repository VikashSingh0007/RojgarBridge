package scrapers

import (
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/PuerkitoBio/goquery"
	"rojgarbridge-backend/modules/job"
	"rojgarbridge-backend/utils"
)

type UPSCScraper struct {
	client *http.Client
}

func NewUPSCScraper() *UPSCScraper {
	return &UPSCScraper{
		client: utils.NewSecureHTTPClient(45 * time.Second),
	}
}

func (s *UPSCScraper) Name() string {
	return "UPSC"
}

func (s *UPSCScraper) BaseURL() string {
	return "https://www.upsc.gov.in"
}

// resolvePDFURL visits the notice landing page on upsc.gov.in and extracts the direct official document PDF link
func (s *UPSCScraper) resolvePDFURL(landingURL string) string {
	if strings.HasSuffix(strings.ToLower(landingURL), ".pdf") {
		return landingURL
	}

	cleanURL := strings.ReplaceAll(landingURL, " ", "%20")
	if !strings.HasPrefix(cleanURL, "http://") && !strings.HasPrefix(cleanURL, "https://") {
		cleanURL = "https://www.upsc.gov.in/" + strings.TrimPrefix(cleanURL, "/")
	}

	req, err := http.NewRequest("GET", cleanURL, nil)
	if err != nil {
		return cleanURL
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")

	resp, err := s.client.Do(req)
	if err != nil {
		return cleanURL
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return cleanURL
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return cleanURL
	}

	var foundPDF string

	// 1. Look inside document table rows in section.body-wrapper
	doc.Find("section.body-wrapper table tr").EachWithBreak(func(_ int, tr *goquery.Selection) bool {
		tr.Find("a[href*='.pdf']").EachWithBreak(func(_ int, a *goquery.Selection) bool {
			href, exists := a.Attr("href")
			if exists && strings.Contains(strings.ToLower(href), ".pdf") && !strings.Contains(href, "Common_Mistake") {
				foundPDF = strings.TrimSpace(href)
				return false
			}
			return true
		})
		return foundPDF == ""
	})

	// 2. Look anywhere inside section.body-wrapper
	if foundPDF == "" {
		doc.Find("section.body-wrapper a[href*='.pdf']").EachWithBreak(func(_ int, a *goquery.Selection) bool {
			href, exists := a.Attr("href")
			if exists && strings.Contains(strings.ToLower(href), ".pdf") && !strings.Contains(href, "Common_Mistake") {
				foundPDF = strings.TrimSpace(href)
				return false
			}
			return true
		})
	}

	// 3. Fallback to any PDF link on the page
	if foundPDF == "" {
		doc.Find("a[href*='.pdf']").EachWithBreak(func(_ int, a *goquery.Selection) bool {
			href, exists := a.Attr("href")
			if exists && strings.Contains(strings.ToLower(href), ".pdf") && !strings.Contains(href, "Common_Mistake") {
				foundPDF = strings.TrimSpace(href)
				return false
			}
			return true
		})
	}

	if foundPDF != "" {
		if !strings.HasPrefix(foundPDF, "http://") && !strings.HasPrefix(foundPDF, "https://") {
			foundPDF = "https://www.upsc.gov.in/" + strings.TrimPrefix(foundPDF, "/")
		}
		return foundPDF
	}

	return cleanURL
}

func (s *UPSCScraper) FetchNotices() ([]job.RawNotice, error) {
	endpoints := []struct {
		url    string
		source string
	}{
		{url: "https://www.upsc.gov.in/whats-new", source: "UPSC What's New"},
		{url: "https://www.upsc.gov.in/examinations/active-exams", source: "UPSC Active Exams"},
	}

	type rawCandidate struct {
		title      string
		landingURL string
		dateText   string
		source     string
	}

	var candidates []rawCandidate
	seen := make(map[string]bool)

	for _, ep := range endpoints {
		req, err := http.NewRequest("GET", ep.url, nil)
		if err != nil {
			continue
		}

		req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36")
		req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,*/*;q=0.8")
		req.Header.Set("Accept-Language", "en-US,en;q=0.9,hi;q=0.8")
		req.Header.Set("Connection", "keep-alive")

		resp, err := s.client.Do(req)
		if err != nil {
			continue
		}

		if resp.StatusCode != http.StatusOK {
			resp.Body.Close()
			continue
		}

		doc, err := goquery.NewDocumentFromReader(resp.Body)
		resp.Body.Close()
		if err != nil {
			continue
		}

		doc.Find(".view-content .views-row, table.views-table tr, table tbody tr, section.body-wrapper tr").Each(func(i int, sel *goquery.Selection) {
			linkSel := sel.Find("a").First()
			title := strings.TrimSpace(linkSel.Text())
			href, exists := linkSel.Attr("href")

			if !exists || title == "" {
				return
			}

			// Ensure absolute URL
			if !strings.HasPrefix(href, "http://") && !strings.HasPrefix(href, "https://") {
				href = "https://www.upsc.gov.in/" + strings.TrimPrefix(href, "/")
			}

			lowerTitle := strings.ToLower(title)
			isJobRelated := strings.Contains(lowerTitle, "recruitment") ||
				strings.Contains(lowerTitle, "examination") ||
				strings.Contains(lowerTitle, "vacancy") ||
				strings.Contains(lowerTitle, "post") ||
				strings.Contains(lowerTitle, "notification") ||
				strings.Contains(lowerTitle, "cgl") ||
				strings.Contains(lowerTitle, "nda") ||
				strings.Contains(lowerTitle, "cds") ||
				strings.Contains(lowerTitle, "civil services") ||
				strings.Contains(lowerTitle, "cse") ||
				strings.Contains(lowerTitle, "admit") ||
				strings.Contains(lowerTitle, "result") ||
				strings.Contains(lowerTitle, "interview") ||
				strings.Contains(lowerTitle, "advt") ||
				strings.Contains(lowerTitle, "marks") ||
				strings.Contains(lowerTitle, "notice")

			if !isJobRelated {
				return
			}

			if seen[href] {
				return
			}
			seen[href] = true

			dateText := strings.TrimSpace(sel.Find(".views-field-created, td:last-child").Text())
			candidates = append(candidates, rawCandidate{
				title:      title,
				landingURL: href,
				dateText:   dateText,
				source:     ep.source,
			})
		})
	}

	// Resolve direct official document PDFs concurrently with worker pool
	type result struct {
		notice job.RawNotice
		index  int
	}

	jobsChan := make(chan struct {
		cand  rawCandidate
		index int
	}, len(candidates))

	resultsChan := make(chan result, len(candidates))

	workerCount := 6
	var wg sync.WaitGroup
	for w := 0; w < workerCount; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for item := range jobsChan {
				pdfURL := s.resolvePDFURL(item.cand.landingURL)
				n := job.RawNotice{
					SourcePortal: "UPSC",
					NoticeTitle:  item.cand.title,
					NoticeURL:    pdfURL,
					PublishedAt:  item.cand.dateText,
					FetchedAt:    time.Now(),
					ContentText:  fmt.Sprintf("Title: %s | Source: %s | PDF: %s | Web Link: %s", item.cand.title, item.cand.source, pdfURL, item.cand.landingURL),
				}
				resultsChan <- result{notice: n, index: item.index}
			}
		}()
	}

	for i, c := range candidates {
		jobsChan <- struct {
			cand  rawCandidate
			index int
		}{cand: c, index: i}
	}
	close(jobsChan)

	wg.Wait()
	close(resultsChan)

	ordered := make([]job.RawNotice, len(candidates))
	for res := range resultsChan {
		ordered[res.index] = res.notice
	}

	return ordered, nil
}
