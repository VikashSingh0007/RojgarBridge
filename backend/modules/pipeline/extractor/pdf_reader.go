package extractor

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/ledongthuc/pdf"
	"rojgarbridge-backend/utils"
)

// PDFReader provides robust extraction of plain text from official government PDF gazettes.
type PDFReader struct {
	client *http.Client
}

// NewPDFReader initializes a PDF reader with secure HTTP client
func NewPDFReader() *PDFReader {
	return &PDFReader{
		client: utils.NewSecureHTTPClient(45 * time.Second),
	}
}

// ExtractTextFromBytes reads the plain text from the first maxPages of the PDF content.
func (pr *PDFReader) ExtractTextFromBytes(data []byte, maxPages int) (string, error) {
	if len(data) == 0 {
		return "", fmt.Errorf("empty pdf data")
	}

	reader, err := pdf.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return "", fmt.Errorf("failed to initialize pdf reader: %w", err)
	}

	totalPages := reader.NumPage()
	if totalPages <= 0 {
		return "", fmt.Errorf("pdf has 0 pages")
	}

	limit := maxPages
	if limit <= 0 || limit > totalPages {
		limit = totalPages
	}

	var sb strings.Builder
	for pageNum := 1; pageNum <= limit; pageNum++ {
		page := reader.Page(pageNum)
		if page.V.IsNull() {
			continue
		}

		plainText, err := page.GetPlainText(nil)
		if err == nil && len(strings.TrimSpace(plainText)) > 0 {
			sb.WriteString(plainText)
			sb.WriteString("\n")
		} else {
			// Fallback to text element traversal
			for _, textRow := range page.Content().Text {
				sb.WriteString(textRow.S)
				sb.WriteString(" ")
			}
			sb.WriteString("\n")
		}
	}

	raw := sb.String()
	// Normalize duplicate spaces and carriage returns
	clean := regexp.MustCompile(`[ \t]+`).ReplaceAllString(raw, " ")
	clean = regexp.MustCompile(`\n{3,}`).ReplaceAllString(clean, "\n\n")
	return strings.TrimSpace(clean), nil
}

// DownloadAndExtractText downloads a PDF from the given URL and extracts its text up to maxPages.
func (pr *PDFReader) DownloadAndExtractText(pdfURL string, maxPages int) (string, error) {
	req, err := http.NewRequest("GET", pdfURL, nil)
	if err != nil {
		return "", fmt.Errorf("invalid request for %s: %w", pdfURL, err)
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "application/pdf,*/*")

	resp, err := pr.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to fetch pdf %s: %w", pdfURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("pdf server returned status %d for %s", resp.StatusCode, pdfURL)
	}

	// Limit to max 15MB to prevent memory exhaustion
	limitReader := io.LimitReader(resp.Body, 15*1024*1024)
	data, err := io.ReadAll(limitReader)
	if err != nil {
		return "", fmt.Errorf("failed to read pdf body: %w", err)
	}

	return pr.ExtractTextFromBytes(data, maxPages)
}

