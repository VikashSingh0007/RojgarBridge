package extractor

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"rozgarhub-backend/internal/models"
)

type Extractor interface {
	Extract(notice models.RawNotice) (*models.ExtractedJobDetails, error)
}

type GeminiExtractor struct {
	apiKey string
	client *http.Client
}

func NewGeminiExtractor(apiKey string) *GeminiExtractor {
	return &GeminiExtractor{
		apiKey: apiKey,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Gemini API Request/Response structs
type geminiContent struct {
	Parts []geminiPart `json:"parts"`
}

type geminiPart struct {
	Text string `json:"text"`
}

type geminiRequest struct {
	Contents []geminiContent `json:"contents"`
}

type geminiResponse struct {
	Candidates []struct {
		Content struct {
			Parts []struct {
				Text string `json:"text"`
			} `json:"parts"`
		} `json:"content"`
	} `json:"candidates"`
}

func (e *GeminiExtractor) Extract(notice models.RawNotice) (*models.ExtractedJobDetails, error) {
	// If API key is not configured, fall back to heuristic/rule-based parsing
	if strings.TrimSpace(e.apiKey) == "" {
		return e.heuristicFallback(notice), nil
	}

	prompt := fmt.Sprintf(`
You are an expert parser for Indian government jobs (Sarkari Naukri).
Given this raw government notice text and metadata:
Portal: %s
Title: %s
URL: %s
Raw Content: %s

Extract structured recruitment details strictly in valid JSON matching this exact schema:
{
  "title": "Clean standard job title (e.g., SSC CGL Examination 2026)",
  "notice_type": "recruitment / result / admit_card / interview_schedule",
  "organization": "e.g., Staff Selection Commission / UPSC / Indian Railways",
  "category": "Central Govt / State Govt / Banking / Defence / Railway / Teaching",
  "state": "All India or specific state",
  "advt_number": "Notification or Advt number if present",
  "total_vacancies": 0,
  "min_age": 18,
  "max_age": 30,
  "qualification": "e.g., 10th Pass / 12th Pass / Graduate in any discipline / B.E/B.Tech",
  "application_start": "YYYY-MM-DD or date string",
  "application_end": "YYYY-MM-DD or date string",
  "exam_date": "Exam date or month if mentioned",
  "fee_general": "e.g., ₹100 or Free",
  "fee_reserved": "e.g., ₹0 for SC/ST/Women",
  "salary_payscale": "Pay level or salary range if found",
  "official_apply_url": "Apply link or portal link",
  "official_notice_url": "Direct PDF or notice page link",
  "summary_points": ["Key point 1", "Key point 2"],
  "confidence_score": 0.95
}

Return ONLY the raw JSON object. Do not include markdown codeblocks or explanation.
`, notice.SourcePortal, notice.NoticeTitle, notice.NoticeURL, notice.ContentText)

	reqPayload := geminiRequest{
		Contents: []geminiContent{
			{
				Parts: []geminiPart{
					{Text: prompt},
				},
			},
		},
	}

	reqBytes, err := json.Marshal(reqPayload)
	if err != nil {
		return nil, err
	}

	endpoint := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/gemini-1.5-flash:generateContent?key=%s", e.apiKey)
	req, err := http.NewRequest("POST", endpoint, bytes.NewBuffer(reqBytes))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := e.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("gemini request error: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		details := e.heuristicFallback(notice)
		details.SummaryPoints = append(details.SummaryPoints, fmt.Sprintf("Note: Rule-based extraction (API %d: %s)", resp.StatusCode, string(respBody)))
		return details, nil
	}

	var geminiResp geminiResponse
	if err := json.NewDecoder(resp.Body).Decode(&geminiResp); err != nil {
		return nil, err
	}

	if len(geminiResp.Candidates) == 0 || len(geminiResp.Candidates[0].Content.Parts) == 0 {
		return e.heuristicFallback(notice), nil
	}

	rawResult := geminiResp.Candidates[0].Content.Parts[0].Text
	// Strip markdown blocks if returned
	rawResult = strings.TrimPrefix(rawResult, "```json")
	rawResult = strings.TrimPrefix(rawResult, "```")
	rawResult = strings.TrimSuffix(rawResult, "```")
	rawResult = strings.TrimSpace(rawResult)

	var details models.ExtractedJobDetails
	if err := json.Unmarshal([]byte(rawResult), &details); err != nil {
		// Fallback if AI produced invalid JSON
		return e.heuristicFallback(notice), nil
	}

	if details.OfficialNoticeURL == "" {
		details.OfficialNoticeURL = notice.NoticeURL
	}

	return &details, nil
}

// heuristicFallback parses common patterns with regex when AI key is missing or quota is exhausted
func (e *GeminiExtractor) heuristicFallback(notice models.RawNotice) *models.ExtractedJobDetails {
	minAge := 18
	maxAge := 30

	// Regex for vacancies like "1200 Vacancies" or "Posts: 450"
	vacancies := 0
	vacRegex := regexp.MustCompile(`(?i)(\d{1,6})\s*(?:vacancies|posts|seats)`)
	if match := vacRegex.FindStringSubmatch(notice.NoticeTitle + " " + notice.ContentText); len(match) > 1 {
		if v, err := strconv.Atoi(match[1]); err == nil {
			vacancies = v
		}
	}

	category := "Central Govt"
	portalUpper := strings.ToUpper(notice.SourcePortal)
	titleUpper := strings.ToUpper(notice.NoticeTitle)

	if portalUpper == "UPSC" || portalUpper == "SSC" || strings.Contains(portalUpper, "RAILWAY") || strings.Contains(portalUpper, "DEFENCE") {
		category = "Central Govt"
	} else if strings.Contains(portalUpper, "IBPS") || strings.Contains(portalUpper, "BANK") || strings.Contains(titleUpper, "BANK") {
		category = "Banking"
	} else if strings.Contains(portalUpper, "PSC") && portalUpper != "UPSC" {
		category = "State Govt"
	}

	// Classify notice type (recruitment, result, admit_card, interview_schedule)
	lowerTitle := strings.ToLower(notice.NoticeTitle)
	noticeType := "recruitment"
	if strings.Contains(lowerTitle, "result") || strings.Contains(lowerTitle, "marks of") || strings.Contains(lowerTitle, "recommended candidates") {
		noticeType = "result"
	} else if strings.Contains(lowerTitle, "interview") || strings.Contains(lowerTitle, "time table") || strings.Contains(lowerTitle, "schedule") {
		noticeType = "interview_schedule"
	} else if strings.Contains(lowerTitle, "admit card") || strings.Contains(lowerTitle, "hall ticket") || strings.Contains(lowerTitle, "e-admit") {
		noticeType = "admit_card"
	}

	return &models.ExtractedJobDetails{
		Title:             notice.NoticeTitle,
		NoticeType:        noticeType,
		Organization:      notice.SourcePortal,
		Category:          category,
		State:             "All India",
		AdvtNumber:        notice.AdvtNumber,
		TotalVacancies:    vacancies,
		MinAge:            &minAge,
		MaxAge:            &maxAge,
		Qualification:     "Refer to official notification",
		ApplicationStart:  notice.PublishedAt,
		ApplicationEnd:    "Refer notification",
		ExamDate:          "To be announced",
		FeeGeneral:        "Refer notification",
		FeeReserved:       "Refer notification",
		SalaryPayScale:    "As per Govt Rules",
		OfficialApplyURL:  notice.NoticeURL,
		OfficialNoticeURL: notice.NoticeURL,
		SummaryPoints: []string{
			fmt.Sprintf("Recruitment notice from %s", notice.SourcePortal),
			"Detailed eligibility & exam pattern available in official PDF",
		},
		ConfidenceScore: 0.70, // Rule-based score
	}
}
