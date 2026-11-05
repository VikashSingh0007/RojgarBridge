package extractor

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"rojgarbridge-backend/modules/job"
)

type OllamaExtractor struct {
	endpoint  string
	model     string
	client    *http.Client
	pdfReader *PDFReader
	ruleBased *RuleBasedExtractor
}

func NewOllamaExtractor(endpoint, model string) *OllamaExtractor {
	if endpoint == "" {
		endpoint = "http://localhost:11434"
	}
	if model == "" {
		model = "llama3.2:3b"
	}
	return &OllamaExtractor{
		endpoint:  strings.TrimRight(endpoint, "/"),
		model:     model,
		client:    &http.Client{Timeout: 90 * time.Second},
		pdfReader: NewPDFReader(),
		ruleBased: NewRuleBasedExtractor(),
	}
}

// IsAvailable checks if Ollama local daemon is online
func (o *OllamaExtractor) IsAvailable() bool {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", o.endpoint+"/api/tags", nil)
	if err != nil {
		return false
	}
	resp, err := o.client.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

type ollamaGenerateRequest struct {
	Model   string                 `json:"model"`
	Prompt  string                 `json:"prompt"`
	Format  string                 `json:"format"`
	Stream  bool                   `json:"stream"`
	Options map[string]interface{} `json:"options"`
}

type ollamaGenerateResponse struct {
	Response string `json:"response"`
	Done     bool   `json:"done"`
}

type ollamaExtractionOutput struct {
	Title            string `json:"title"`
	NoticeType       string `json:"notice_type"`
	Organization     string `json:"organization"`
	Category         string `json:"category"`
	State            string `json:"state"`
	AdvtNumber       string `json:"advt_number"`
	TotalVacancies   int    `json:"total_vacancies"`
	MinAge           *int   `json:"min_age"`
	MaxAge           *int   `json:"max_age"`
	Qualification    string `json:"qualification"`
	ApplicationStart string `json:"application_start"`
	ApplicationEnd   string `json:"application_end"`
	ExamDate         string `json:"exam_date"`
	FeeGeneral       string `json:"fee_general"`
	FeeReserved      string `json:"fee_reserved"`
	SalaryPayScale   string `json:"salary_payscale"`
}

func (o *OllamaExtractor) Extract(notice job.RawNotice) (*job.ExtractedJobDetails, error) {
	nType := ExtractNoticeType(notice.NoticeTitle + " " + notice.ContentText)
	hasPDF := strings.Contains(strings.ToLower(notice.NoticeURL), ".pdf")

	// 1. Announcements (Answer Keys, Results, Admit Cards, Exam Dates, Corrigenda) do not have application vacancies.
	// Deterministic RuleBasedExtractor classifies them instantly (0.1ms) with 100% accuracy.
	if nType != "recruitment" {
		return o.ruleBased.Extract(notice)
	}

	// 2. For Recruitment Openings, extract plain text from first 3 pages of the official PDF
	pdfText := ""
	if hasPDF {
		text, err := o.pdfReader.DownloadAndExtractText(notice.NoticeURL, 3)
		if err == nil && len(text) > 0 {
			pdfText = text
		}
	}

	// 3. Try Ollama local LLM for recruitment metadata extraction
	if o.IsAvailable() {
		details, err := o.extractWithOllama(notice, pdfText)
		if err == nil && details != nil {
			return details, nil
		}
		log.Printf("[Ollama] Extraction failed (%v), falling back to deterministic RuleBasedExtractor", err)
	}

	// 4. Fallback to RuleBasedExtractor with enriched PDF context
	enrichedNotice := notice
	if pdfText != "" {
		enrichedNotice.ContentText = notice.ContentText + " | PDF Excerpt: " + pdfText
	}
	return o.ruleBased.Extract(enrichedNotice)
}

func (o *OllamaExtractor) extractWithOllama(notice job.RawNotice, pdfText string) (*job.ExtractedJobDetails, error) {
	contextData := fmt.Sprintf("Source Portal: %s\nNotice Title: %s\nPublished Date: %s\nNotice URL: %s\nSummary: %s",
		notice.SourcePortal, notice.NoticeTitle, notice.PublishedAt, notice.NoticeURL, notice.ContentText)

	if len(pdfText) > 0 {
		maxSnippet := 3500
		if len(pdfText) < maxSnippet {
			maxSnippet = len(pdfText)
		}
		contextData += fmt.Sprintf("\nOfficial PDF Content (first pages):\n%s", pdfText[:maxSnippet])
	}

	prompt := fmt.Sprintf(`You are an expert government job notice parser in India.
Extract structured metadata from the following official government announcement:

%s

Respond ONLY with a JSON object matching this exact schema:
{
  "title": "Clean job or announcement title (string)",
  "notice_type": "recruitment | result | admit_card | answer_key | interview_schedule | corrigendum",
  "organization": "Exact hiring authority name, e.g. Staff Selection Commission (SSC) or Institute of Banking Personnel Selection (IBPS)",
  "category": "Central Govt | Banking | Railway | Defence & Police | Teaching | Engineering & PSU | Medical & Healthcare | Judiciary | State Govt",
  "state": "All India or Indian State name",
  "advt_number": "Official advertisement / notification number or empty",
  "total_vacancies": 0,
  "min_age": 18,
  "max_age": 30,
  "qualification": "10th Pass | 12th Pass | Graduate | B.E / B.Tech | ITI | Diploma | Master Degree | Refer to official notification",
  "application_start": "YYYY-MM-DD or empty",
  "application_end": "YYYY-MM-DD or Refer notification",
  "exam_date": "Exam date or To be announced",
  "fee_general": "Fee amount e.g. ₹100 or ₹850 or Exempted or Refer official notification",
  "fee_reserved": "Fee for SC/ST/PwBD/Female or Exempted",
  "salary_payscale": "Pay level or monthly salary scale or empty"
}
Rules:
1. If vacancies are not stated, set total_vacancies to 0.
2. If notice is a result, admit card, answer key, or date schedule, set notice_type accordingly.
3. For dates, format as YYYY-MM-DD whenever possible.
4. Do not invent facts. Return valid JSON only.`, contextData)

	reqBody := ollamaGenerateRequest{
		Model:  o.model,
		Prompt: prompt,
		Format: "json",
		Stream: false,
		Options: map[string]interface{}{
			"temperature": 0.1,
			"top_p":       0.9,
		},
	}

	payload, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	httpReq, err := http.NewRequestWithContext(ctx, "POST", o.endpoint+"/api/generate", bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := o.client.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ollama status code %d", resp.StatusCode)
	}

	var genResp ollamaGenerateResponse
	if err := json.NewDecoder(resp.Body).Decode(&genResp); err != nil {
		return nil, fmt.Errorf("failed to decode ollama response: %w", err)
	}

	var out ollamaExtractionOutput
	if err := json.Unmarshal([]byte(genResp.Response), &out); err != nil {
		return nil, fmt.Errorf("failed to unmarshal ollama json output: %w (raw: %s)", err, genResp.Response)
	}

	if out.Title == "" {
		out.Title = CleanJobTitle(notice.NoticeTitle)
	}
	if out.Organization == "" {
		out.Organization = ExtractRealOrg(notice.NoticeTitle, notice.ContentText, notice.SourcePortal)
	}
	if out.Category == "" {
		out.Category = ExtractCategory(notice.SourcePortal, out.Organization, notice.NoticeTitle)
	}
	if out.State == "" {
		out.State = "All India"
	}
	if out.ApplicationEnd == "" {
		out.ApplicationEnd = "Refer notification"
	}
	if out.ExamDate == "" {
		out.ExamDate = "To be announced"
	}
	if out.FeeGeneral == "" {
		out.FeeGeneral = "Refer official notification"
	}
	if out.FeeReserved == "" {
		out.FeeReserved = "Refer official notification"
	}

	officialURL := ExtractOfficialGovURL(notice.ContentText+" "+notice.NoticeTitle, notice.NoticeURL)
	summaryPoints := GenerateSummaryPoints(out.Organization, out.Title, out.NoticeType, out.TotalVacancies, out.Qualification, out.ApplicationEnd, out.State, officialURL)

	return &job.ExtractedJobDetails{
		Title:              out.Title,
		NoticeType:         out.NoticeType,
		Organization:       out.Organization,
		Category:           out.Category,
		State:              out.State,
		AdvtNumber:         out.AdvtNumber,
		TotalVacancies:     out.TotalVacancies,
		MinAge:             out.MinAge,
		MaxAge:             out.MaxAge,
		Qualification:      out.Qualification,
		ApplicationStart:   out.ApplicationStart,
		ApplicationEnd:     out.ApplicationEnd,
		ExamDate:           out.ExamDate,
		FeeGeneral:         out.FeeGeneral,
		FeeReserved:        out.FeeReserved,
		SalaryPayScale:     out.SalaryPayScale,
		OfficialApplyURL:   officialURL,
		OfficialNoticeURL:  notice.NoticeURL,
		SummaryPoints:      summaryPoints,
		ConfidenceScore:    0.95,
	}, nil
}

