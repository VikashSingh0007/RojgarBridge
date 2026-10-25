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

	"rojgarbridge-backend/internal/models"
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
  "title": "Clean standard job title (e.g., SSC CGL Examination 2026 or PNB Rohtak Faculty & Watchman Recruitment 2026)",
  "notice_type": "recruitment / result / admit_card / interview_schedule / answer_key",
  "organization": "e.g., Staff Selection Commission / Union Bank of India / Indian Railways / UPSC",
  "category": "Central Govt / State Govt / Banking / Defence & Police / Railway / Teaching / Engineering & PSU / Medical & Healthcare / Judiciary",
  "state": "State name (e.g., Uttar Pradesh, Bihar, Haryana, West Bengal) or 'All India'",
  "advt_number": "Notification or Advt number if present",
  "total_vacancies": 0,
  "min_age": 18,
  "max_age": 30,
  "qualification": "Extracted educational qualification e.g., 10th Pass / Any Graduate / B.E/B.Tech / 12th Pass",
  "application_start": "YYYY-MM-DD or date string",
  "application_end": "YYYY-MM-DD or date string",
  "exam_date": "Exam date or month if mentioned",
  "fee_general": "e.g., ₹100 or Free / Nil",
  "fee_reserved": "e.g., ₹0 for SC/ST/Women",
  "salary_payscale": "Pay level or salary range e.g., Rs. 25,000 - 80,000 or Level-7",
  "official_apply_url": "Apply link or portal link",
  "official_notice_url": "Direct PDF or notice page link",
  "summary_points": ["Key point 1", "Key point 2", "Key point 3"],
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
		details.SummaryPoints = append(details.SummaryPoints, fmt.Sprintf("Note: Smart heuristic extraction (API %d: %s)", resp.StatusCode, string(respBody)))
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
	rawResult = strings.TrimPrefix(rawResult, "```json")
	rawResult = strings.TrimPrefix(rawResult, "```")
	rawResult = strings.TrimSuffix(rawResult, "```")
	rawResult = strings.TrimSpace(rawResult)

	var details models.ExtractedJobDetails
	if err := json.Unmarshal([]byte(rawResult), &details); err != nil {
		return e.heuristicFallback(notice), nil
	}

	details.OfficialNoticeURL = extractOfficialGovURL(details.OfficialNoticeURL+" "+notice.ContentText, details.OfficialNoticeURL)
	details.OfficialApplyURL = extractOfficialGovURL(details.OfficialApplyURL+" "+notice.ContentText, details.OfficialApplyURL)

	return &details, nil
}

// heuristicFallback parses structured patterns with deep regex & dictionary lookup
func (e *GeminiExtractor) heuristicFallback(notice models.RawNotice) *models.ExtractedJobDetails {
	fullText := notice.NoticeTitle + " " + notice.ContentText
	lowerText := strings.ToLower(fullText)

	// 1. Notice Type Detection
	noticeType := "recruitment"
	if strings.Contains(lowerText, "admit card") || strings.Contains(lowerText, "hall ticket") || strings.Contains(lowerText, "call letter") || strings.Contains(lowerText, "e-admit") {
		noticeType = "admit_card"
	} else if strings.Contains(lowerText, "result") || strings.Contains(lowerText, "marks of") || strings.Contains(lowerText, "recommended candidates") || strings.Contains(lowerText, "merit list") || strings.Contains(lowerText, "score card") || strings.Contains(lowerText, "cut-off") {
		noticeType = "result"
	} else if strings.Contains(lowerText, "answer key") || strings.Contains(lowerText, "response sheet") || strings.Contains(lowerText, "question paper") {
		noticeType = "answer_key"
	} else if strings.Contains(lowerText, "interview") || strings.Contains(lowerText, "walkin") || strings.Contains(lowerText, "walk-in") || strings.Contains(lowerText, "time table") || strings.Contains(lowerText, "schedule") {
		noticeType = "interview_schedule"
	}

	// 2. Organization Extraction
	orgName := extractRealOrg(notice.NoticeTitle, notice.ContentText, notice.SourcePortal)

	// 3. Category Detection
	category := extractCategory(notice.SourcePortal, orgName, fullText)

	// 4. State Detection
	state := extractIndianState(notice.NoticeTitle, notice.ContentText, orgName, category)

	// 5. Total Vacancies Extraction
	vacancies := extractVacancies(notice.NoticeTitle, notice.ContentText)

	// 6. Qualification Extraction
	qualification := extractQualification(notice.ContentText, notice.NoticeTitle)

	// 7. Last Date / Application End
	appEnd := extractLastDate(notice.ContentText, notice.PublishedAt)

	// 8. Application Start Date
	appStart := notice.PublishedAt
	if match := regexp.MustCompile(`(?i)(?:start(?:ing)?\s*date|apply\s*online\s*from)\s*[:–-]?\s*([0-3]?\d[-/.][0-1]?\d[-/.]\d{2,4})`).FindStringSubmatch(notice.ContentText); len(match) > 1 {
		appStart = match[1]
	}

	// 9. Age Limits
	minAge, maxAge := extractAgeLimits(fullText)

	// 10. Fee details
	feeGen, feeRes := extractFees(fullText)

	// 11. Salary / Pay Scale
	salary := extractSalary(fullText)

	// 12. Advt Number
	advtNo := notice.AdvtNumber
	if advtNo == "" {
		if match := regexp.MustCompile(`(?i)(?:advt\.?\s*no\.?|notification\s*no\.?)\s*[:–-]?\s*([a-zA-Z0-9/._-]+)`).FindStringSubmatch(fullText); len(match) > 1 {
			advtNo = match[1]
		}
	}

	// 13. Official Government Portal URL
	officialURL := extractOfficialGovURL(notice.ContentText+" "+notice.NoticeTitle, notice.NoticeURL)

	// 14. Rich Summary Points
	summaryPoints := generateSummaryPoints(orgName, noticeType, vacancies, qualification, appEnd, state, officialURL)

	cleanTitle := cleanJobTitle(notice.NoticeTitle)

	confidence := 0.88
	if qualification != "Refer to official notification" && appEnd != "Refer notification" {
		confidence = 0.95
	}

	return &models.ExtractedJobDetails{
		Title:             cleanTitle,
		NoticeType:        noticeType,
		Organization:      orgName,
		Category:          category,
		State:             state,
		AdvtNumber:        advtNo,
		TotalVacancies:    vacancies,
		MinAge:            minAge,
		MaxAge:            maxAge,
		Qualification:     qualification,
		ApplicationStart:  appStart,
		ApplicationEnd:    appEnd,
		ExamDate:          extractExamDate(fullText),
		FeeGeneral:        feeGen,
		FeeReserved:       feeRes,
		SalaryPayScale:    salary,
		OfficialApplyURL:  officialURL,
		OfficialNoticeURL: officialURL,
		SummaryPoints:     summaryPoints,
		ConfidenceScore:   confidence,
	}
}

// Extract Category accurately
func extractCategory(sourcePortal, orgName, text string) string {
	lowerPortal := strings.ToLower(sourcePortal)
	lowerOrg := strings.ToLower(orgName)
	lowerText := strings.ToLower(text)

	if strings.Contains(lowerPortal, "bank") || strings.Contains(lowerOrg, "bank") || strings.Contains(lowerOrg, "ibps") || strings.Contains(lowerOrg, "sbi") || strings.Contains(lowerOrg, "rbi") || strings.Contains(lowerOrg, "nabard") || strings.Contains(lowerOrg, "sebi") {
		return "Banking"
	}
	if strings.Contains(lowerPortal, "railway") || strings.Contains(lowerOrg, "railway") || strings.Contains(lowerOrg, "rrb") || strings.Contains(lowerOrg, "rrc") || strings.Contains(lowerOrg, "metro") || strings.Contains(lowerText, "irctc") {
		return "Railway"
	}
	if strings.Contains(lowerPortal, "police") || strings.Contains(lowerPortal, "defence") || strings.Contains(lowerOrg, "police") || strings.Contains(lowerOrg, "army") || strings.Contains(lowerOrg, "navy") || strings.Contains(lowerOrg, "air force") || strings.Contains(lowerOrg, "crpf") || strings.Contains(lowerOrg, "bsf") || strings.Contains(lowerOrg, "cisf") || strings.Contains(lowerOrg, "itbp") || strings.Contains(lowerOrg, "assam rifles") || strings.Contains(lowerText, "constable") || strings.Contains(lowerText, "sub inspector") {
		return "Defence & Police"
	}
	if strings.Contains(lowerPortal, "teaching") || strings.Contains(lowerOrg, "university") || strings.Contains(lowerOrg, "college") || strings.Contains(lowerText, "professor") || strings.Contains(lowerText, "teacher") || strings.Contains(lowerText, "faculty") || strings.Contains(lowerText, "ctet") || strings.Contains(lowerText, "ugc net") {
		return "Teaching"
	}
	if strings.Contains(lowerPortal, "engineering") || strings.Contains(lowerText, "gate") || strings.Contains(lowerOrg, "bhel") || strings.Contains(lowerOrg, "ntpc") || strings.Contains(lowerOrg, "ongc") || strings.Contains(lowerOrg, "iocl") || strings.Contains(lowerOrg, "drdo") || strings.Contains(lowerOrg, "isro") || strings.Contains(lowerOrg, "sail") || strings.Contains(lowerOrg, "bel") {
		return "Engineering & PSU"
	}
	if strings.Contains(lowerOrg, "hospital") || strings.Contains(lowerOrg, "aiims") || strings.Contains(lowerOrg, "medical") || strings.Contains(lowerText, "resident doctor") || strings.Contains(lowerText, "staff nurse") || strings.Contains(lowerText, "pharmacist") || strings.Contains(lowerText, "mbbs") || strings.Contains(lowerText, "nhm") {
		return "Medical & Healthcare"
	}
	if strings.Contains(lowerOrg, "court") || strings.Contains(lowerText, "judiciary") || strings.Contains(lowerText, "civil judge") || strings.Contains(lowerText, "law officer") {
		return "Judiciary"
	}
	if strings.Contains(lowerPortal, "state") || strings.Contains(lowerOrg, "psc") && !strings.Contains(lowerOrg, "upsc") {
		return "State Govt"
	}
	if strings.Contains(lowerPortal, "upsc") || strings.Contains(lowerPortal, "ssc") || strings.Contains(lowerPortal, "central") {
		return "Central Govt"
	}

	return "Central Govt"
}

// Indian State & UT dictionary detection
func extractIndianState(title, content, org, category string) string {
	combined := strings.ToLower(title + " " + content + " " + org)

	stateMap := map[string][]string{
		"Andhra Pradesh":    {"andhra pradesh", "ap psc", "appsc", "amaravati", "visakhapatnam", "vizag", "vijayawada", "guntur", "tirupati"},
		"Arunachal Pradesh": {"arunachal pradesh", "itanagar", "appsc itanagar"},
		"Assam":             {"assam", "guwahati", "dispur", "apsc", "silchar", "dibrugarh", "tezpur"},
		"Bihar":             {"bihar", "patna", "bpsc", "bssc", "gaya", "muzaffarpur", "bhagalpur", "darbhanga", "nalanda"},
		"Chhattisgarh":      {"chhattisgarh", "raipur", "bilaspur", "cgpsc", "durg", "bhilai", "bastar"},
		"Delhi":             {"delhi", "new delhi", "dsssb", "kaushik enclave", "burari", "delhi cantonment", "nct"},
		"Goa":               {"goa", "panaji", "margao", "vasco", "gpsc goa"},
		"Gujarat":           {"gujarat", "gandhinagar", "ahmedabad", "surat", "vadodara", "rajkot", "gpsc"},
		"Haryana":           {"haryana", "rohtak", "gurugram", "gurgaon", "faridabad", "panipat", "ambala", "hpsc", "hssc", "haryana police"},
		"Himachal Pradesh":  {"himachal pradesh", "shimla", "dharamshala", "mandi", "hppsc", "hpsssb"},
		"Jharkhand":         {"jharkhand", "ranchi", "jamshedpur", "dhanbad", "bokaro", "jpsc", "jssc"},
		"Karnataka":         {"karnataka", "bengaluru", "bangalore", "mysore", "mysuru", "hubli", "kpsc", "dharwad", "mangalore"},
		"Kerala":            {"kerala", "thiruvananthapuram", "kochi", "cochin", "kozhikode", "calicut", "thrissur", "kpsc kerala"},
		"Madhya Pradesh":    {"madhya pradesh", "bhopal", "indore", "gwalior", "jabalpur", "ujjain", "mppsc"},
		"Maharashtra":       {"maharashtra", "mumbai", "pune", "nagpur", "nashik", "thane", "aurangabad", "mpsc"},
		"Manipur":           {"manipur", "imphal"},
		"Meghalaya":         {"meghalaya", "shillong"},
		"Mizoram":           {"mizoram", "aizawl"},
		"Nagaland":          {"nagaland", "kohima", "dimapur"},
		"Odisha":            {"odisha", "orissa", "bhubaneswar", "cuttack", "rourkela", "opsc", "osssc", "sambalpur"},
		"Punjab":            {"punjab", "ludhiana", "amritsar", "jalandhar", "patiala", "bathinda", "ppsc", "psssb"},
		"Rajasthan":         {"rajasthan", "jaipur", "jodhpur", "udaipur", "kota", "ajmer", "bikaner", "rpsc", "rsmssb"},
		"Sikkim":            {"sikkim", "gangtok"},
		"Tamil Nadu":        {"tamil nadu", "tamilnadu", "chennai", "coimbatore", "madurai", "tnpsc", "salem", "trichy", "tiruchirappalli"},
		"Telangana":         {"telangana", "hyderabad", "warangal", "tspsc", "secunderabad", "nizamabad"},
		"Tripura":           {"tripura", "agartala"},
		"Uttar Pradesh":     {"uttar pradesh", "lucknow", "kanpur", "varanasi", "prayagraj", "allahabad", "agra", "noida", "meerut", "bahraich", "bareilly", "gorakhpur", "uppsc", "upsssc", "asmc bahraich"},
		"Uttarakhand":       {"uttarakhand", "dehradun", "haridwar", "nainital", "roorkee", "ukpsc", "uksssc"},
		"West Bengal":       {"west bengal", "wb police", "wbpsc", "kolkata", "howrah", "siliguri", "dakshin dinajpur", "asansol", "durgapur", "malda"},
		"Jammu & Kashmir":   {"jammu", "kashmir", "srinagar", "jkssb", "jkpsc"},
		"Chandigarh":        {"chandigarh"},
	}

	for stateName, keywords := range stateMap {
		for _, kw := range keywords {
			if strings.Contains(combined, kw) {
				return stateName
			}
		}
	}

	if category == "Central Govt" || category == "Railway" || category == "Banking" || strings.Contains(combined, "upsc") || strings.Contains(combined, "ssc") {
		return "All India"
	}

	return "All India"
}

// Vacancies extraction
func extractVacancies(title, content string) int {
	combined := title + " " + content

	// Pattern 1: [PNB Rohtak] ... – 2 Posts
	re1 := regexp.MustCompile(`(?i)(?:–|-|:)?\s*(\d{1,6})\s*(?:posts?|vacanc(?:y|ies)|seats?)`)
	if match := re1.FindStringSubmatch(combined); len(match) > 1 {
		if v, err := strconv.Atoi(match[1]); err == nil && v > 0 && v < 500000 {
			return v
		}
	}

	// Pattern 2: "for 70 Driver posts" or "recruitment of 16 Senior Resident"
	re2 := regexp.MustCompile(`(?i)(?:for|of)\s+(\d{1,6})\s+(?:various\s+)?(?:[a-zA-Z\s-]{1,25})\s*(?:posts?|vacanc(?:y|ies))`)
	if match := re2.FindStringSubmatch(combined); len(match) > 1 {
		if v, err := strconv.Atoi(match[1]); err == nil && v > 0 {
			return v
		}
	}

	return 0
}

// Qualification extraction
func extractQualification(content, title string) string {
	// 1. Direct table column "Qualification: Any Graduate, 7th"
	if match := regexp.MustCompile(`(?i)qualification\s*[:–-]\s*([^|]+)`).FindStringSubmatch(content); len(match) > 1 {
		q := strings.TrimSpace(match[1])
		if q != "" && q != "–" && q != "-" {
			return q
		}
	}

	combined := strings.ToLower(title + " " + content)

	var qualifications []string
	if strings.Contains(combined, "10th") || strings.Contains(combined, "matric") || strings.Contains(combined, "sslc") {
		qualifications = append(qualifications, "10th Pass")
	}
	if strings.Contains(combined, "12th") || strings.Contains(combined, "10+2") || strings.Contains(combined, "intermediate") || strings.Contains(combined, "higher secondary") {
		qualifications = append(qualifications, "12th Pass")
	}
	if strings.Contains(combined, "iti") {
		qualifications = append(qualifications, "ITI")
	}
	if strings.Contains(combined, "diploma") || strings.Contains(combined, "polytechnic") {
		qualifications = append(qualifications, "Diploma")
	}
	if strings.Contains(combined, "graduate") || strings.Contains(combined, "degree") || strings.Contains(combined, "b.a") || strings.Contains(combined, "b.sc") || strings.Contains(combined, "b.com") {
		qualifications = append(qualifications, "Graduate")
	}
	if strings.Contains(combined, "b.e") || strings.Contains(combined, "b.tech") || strings.Contains(combined, "engineering") {
		qualifications = append(qualifications, "B.E / B.Tech")
	}
	if strings.Contains(combined, "post graduate") || strings.Contains(combined, "master") || strings.Contains(combined, "m.tech") || strings.Contains(combined, "mba") {
		qualifications = append(qualifications, "Post Graduate")
	}
	if strings.Contains(combined, "mbbs") || strings.Contains(combined, "md/ms") || strings.Contains(combined, "bds") {
		qualifications = append(qualifications, "MBBS / Medical Degree")
	}
	if strings.Contains(combined, "b.ed") || strings.Contains(combined, "d.el.ed") {
		qualifications = append(qualifications, "B.Ed / D.El.Ed")
	}
	if strings.Contains(combined, "ca") || strings.Contains(combined, "icwa") {
		qualifications = append(qualifications, "CA / Finance")
	}
	if strings.Contains(combined, "8th") || strings.Contains(combined, "7th") {
		qualifications = append(qualifications, "8th / 7th Pass")
	}

	if len(qualifications) > 0 {
		return strings.Join(qualifications, ", ")
	}

	return "Refer to official notification"
}

// Last Date extraction
func extractLastDate(content, defaultDate string) string {
	// 1. Direct "Last Date: 10-11-2026"
	if match := regexp.MustCompile(`(?i)last\s*date\s*[:–-]\s*([^|]+)`).FindStringSubmatch(content); len(match) > 1 {
		ld := strings.TrimSpace(match[1])
		if ld != "" && ld != "–" && ld != "-" {
			return ld
		}
	}

	// 2. Pattern matching dates like "till 05 November 2026" or "before 15-12-2026"
	dateRe := regexp.MustCompile(`(?i)(?:till|before|reach\s*on\s*or\s*before|closing\s*date\s*is|last\s*date\s*is)\s*([0-3]?\d\s+(?:January|February|March|April|May|June|July|August|September|October|November|December|Jan|Feb|Mar|Apr|May|Jun|Jul|Aug|Sep|Oct|Nov|Dec)[a-z]*\s+\d{4})`)
	if match := dateRe.FindStringSubmatch(content); len(match) > 1 {
		return match[1]
	}

	dateRe2 := regexp.MustCompile(`(?i)(?:till|before|reach\s*on\s*or\s*before|closing\s*date|last\s*date)\s*[:–-]?\s*([0-3]?\d[-/.][0-1]?\d[-/.]\d{2,4})`)
	if match := dateRe2.FindStringSubmatch(content); len(match) > 1 {
		return match[1]
	}

	return "Refer notification"
}

// Age Limits
func extractAgeLimits(text string) (*int, *int) {
	// "between 21 and 40 years" or "21 to 40 years"
	re := regexp.MustCompile(`(?i)(?:between|age\s*limit\s*is|age\s*limit)\s*[:–-]?\s*(\d{2})\s*(?:to|and|-)\s*(\d{2})\s*years?`)
	if match := re.FindStringSubmatch(text); len(match) > 2 {
		minA, err1 := strconv.Atoi(match[1])
		maxA, err2 := strconv.Atoi(match[2])
		if err1 == nil && err2 == nil && minA >= 16 && maxA <= 65 {
			return &minA, &maxA
		}
	}

	defaultMin := 18
	defaultMax := 30
	return &defaultMin, &defaultMax
}

// Fee details
func extractFees(text string) (string, string) {
	lower := strings.ToLower(text)
	if strings.Contains(lower, "no fee") || strings.Contains(lower, "no application fee") || strings.Contains(lower, "free") {
		return "Free / Nil", "Free / Nil"
	}

	if match := regexp.MustCompile(`(?i)(?:fee|application\s*fee)\s*[:–-]?\s*(?:rs\.?|₹)?\s*(\d{1,5})`).FindStringSubmatch(text); len(match) > 1 {
		fee := fmt.Sprintf("₹%s", match[1])
		return fee, "As per rules"
	}

	return "Refer official notification", "Refer official notification"
}

// Salary details
func extractSalary(text string) string {
	if match := regexp.MustCompile(`(?i)(?:monthly\s+salary\s+of\s+|salary\s*[:–-]\s*|pay\s*scale\s*[:–-]\s*)(?:rs\.?|₹)?\s*([\d,]+(?:\s*(?:to|-)\s*[\d,]+)?(?:\s*(?:per\s*month|\/-|p\.m\.))?)`).FindStringSubmatch(text); len(match) > 1 {
		return "₹" + strings.TrimPrefix(strings.TrimSpace(match[1]), "Rs. ")
	}
	if match := regexp.MustCompile(`(?i)(level[- ]\d+|pay\s*level[- ]\d+|grade\s*pay\s*\d+)`).FindStringSubmatch(text); len(match) > 1 {
		return strings.ToUpper(match[1])
	}
	return "As per Govt Rules"
}

// Exam date
func extractExamDate(text string) string {
	if match := regexp.MustCompile(`(?i)(?:interview\s*on|exam\s*on|examination\s*date|exam\s*date)\s*[:–-]?\s*([0-3]?\d\s+(?:January|February|March|April|May|June|July|August|September|October|November|December|Jan|Feb|Mar|Apr|May|Jun|Jul|Aug|Sep|Oct|Nov|Dec)[a-z]*\s+\d{4})`).FindStringSubmatch(text); len(match) > 1 {
		return match[1]
	}
	if match := regexp.MustCompile(`(?i)(?:interview\s*on|exam\s*on|examination\s*date|exam\s*date)\s*[:–-]?\s*([0-3]?\d[-/.][0-1]?\d[-/.]\d{2,4})`).FindStringSubmatch(text); len(match) > 1 {
		return match[1]
	}
	return "To be announced"
}

// Summary points generator
func generateSummaryPoints(org, noticeType string, vacancies int, qual, lastDate, state, officialURL string) []string {
	var points []string

	if vacancies > 0 {
		points = append(points, fmt.Sprintf("%d Total Vacancies announced by %s", vacancies, org))
	} else {
		points = append(points, fmt.Sprintf("Official %s update from %s", strings.ReplaceAll(noticeType, "_", " "), org))
	}

	if qual != "Refer to official notification" {
		points = append(points, fmt.Sprintf("Required Qualification: %s", qual))
	}

	if lastDate != "Refer notification" && lastDate != "" {
		points = append(points, fmt.Sprintf("Application Deadline / Last Date: %s", lastDate))
	}

	if state != "" && state != "All India" {
		points = append(points, fmt.Sprintf("Job Location / State: %s", state))
	}

	if officialURL != "" {
		points = append(points, fmt.Sprintf("Official Portal / Gazette: %s", officialURL))
	}

	return points
}

// Clean title
func cleanJobTitle(title string) string {
	title = strings.ReplaceAll(title, "&#8211;", "–")
	title = strings.ReplaceAll(title, "&#8217;", "'")
	title = strings.ReplaceAll(title, "&amp;", "&")
	title = strings.TrimSpace(title)
	return title
}

// extractOfficialGovURL extracts real .gov.in, .nic.in, .ac.in, or .edu.in domains from text
func extractOfficialGovURL(text string, fallbackURL string) string {
	lowerFallback := strings.ToLower(fallbackURL)
	if strings.Contains(lowerFallback, ".gov.in") ||
		strings.Contains(lowerFallback, ".nic.in") ||
		strings.Contains(lowerFallback, ".ac.in") ||
		strings.Contains(lowerFallback, ".edu.in") {
		return fallbackURL
	}

	govRegex := regexp.MustCompile(`(?i)(?:https?://)?([a-z0-9.-]+\.(?:gov\.in|nic\.in|ac\.in|edu\.in|org\.in|res\.in|mil\.in))`)
	if match := govRegex.FindStringSubmatch(text); len(match) > 1 {
		domain := strings.ToLower(match[1])
		domain = strings.TrimSuffix(domain, ".")
		return "https://" + domain
	}

	// Never return 3rd-party aggregator links
	if strings.Contains(lowerFallback, "freejobalert.com") ||
		strings.Contains(lowerFallback, "sarkariresult") ||
		strings.Contains(lowerFallback, "blogspot") {
		return ""
	}

	return fallbackURL
}

// extractRealOrg extracts authentic organization name
func extractRealOrg(title, content, defaultPortal string) string {
	// Check "Board: PNB Bank Rohtak" in structured content
	if match := regexp.MustCompile(`(?i)board\s*[:–-]\s*([^|]+)`).FindStringSubmatch(content); len(match) > 1 {
		b := strings.TrimSpace(match[1])
		if b != "" && b != "–" && b != "-" {
			return b
		}
	}

	// Check [PNB Bank Rohtak] in title
	if strings.HasPrefix(title, "[") && strings.Contains(title, "]") {
		bracket := title[1:strings.Index(title, "]")]
		bracket = strings.TrimSpace(bracket)
		if bracket != "" {
			return bracket
		}
	}

	// Common official names
	if defaultPortal == "UPSC" {
		return "Union Public Service Commission (UPSC)"
	}
	if defaultPortal == "SSC" {
		return "Staff Selection Commission (SSC)"
	}
	if defaultPortal == "IBPS" {
		return "Institute of Banking Personnel Selection (IBPS)"
	}
	if defaultPortal == "RRB" {
		return "Railway Recruitment Board (RRB)"
	}

	// Organization before "Recruitment"
	titleParts := strings.Split(title, "Recruitment")
	if len(titleParts) > 1 && len(strings.TrimSpace(titleParts[0])) > 2 {
		candidate := strings.TrimSpace(titleParts[0])
		candidate = strings.TrimPrefix(candidate, "[")
		candidate = strings.TrimSuffix(candidate, "]")
		candidate = strings.TrimSuffix(candidate, "–")
		candidate = strings.TrimSuffix(candidate, "-")
		return strings.TrimSpace(candidate)
	}

	// Check content like "Inland Waterways Authority of India (IWAI) has released..."
	orgRegex := regexp.MustCompile(`^([^()]+(?:\([^)]+\))?)\s+(?:has\s+released|has\s+published|has\s+invited|is\s+conducting)`)
	if match := orgRegex.FindStringSubmatch(content); len(match) > 1 {
		return strings.TrimSpace(match[1])
	}

	return defaultPortal
}
