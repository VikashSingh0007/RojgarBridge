package extractor

import (
	"fmt"
	"net/url"
	"regexp"
	"strconv"
	"strings"

	"rojgarbridge-backend/modules/job"
)

type Extractor interface {
	Extract(notice job.RawNotice) (*job.ExtractedJobDetails, error)
}

type RuleBasedExtractor struct{}

func NewRuleBasedExtractor() *RuleBasedExtractor {
	return &RuleBasedExtractor{}
}

func (e *RuleBasedExtractor) Extract(notice job.RawNotice) (*job.ExtractedJobDetails, error) {
	fullText := notice.NoticeTitle + " " + notice.ContentText
	cleanTitle := CleanJobTitle(notice.NoticeTitle)

	// 1. Notice Type Detection
	noticeType := ExtractNoticeType(fullText)

	// 2. Organization Extraction
	orgName := ExtractRealOrg(notice.NoticeTitle, notice.ContentText, notice.SourcePortal)

	// 3. Category Detection
	category := ExtractCategory(notice.SourcePortal, orgName, fullText)

	// 4. State Detection
	state := ExtractIndianState(notice.NoticeTitle, notice.ContentText, orgName, category)

	// 5. Details Extraction (Parse only what exists; never fabricate)
	var vacancies int
	var qualification string
	var minAge, maxAge *int
	var appStart, appEnd, examDate string
	var feeGen, feeRes string
	var salary string

	if noticeType == "recruitment" {
		vacancies = ExtractVacancies(notice.NoticeTitle, notice.ContentText)
		qualification = ExtractQualification(notice.ContentText, notice.NoticeTitle)
		minAge, maxAge = ExtractAgeLimits(fullText)
		appStart, appEnd, examDate = ExtractDates(fullText, notice.PublishedAt)
		feeGen, feeRes = ExtractFees(fullText)
		salary = ExtractSalary(fullText)
	} else {
		// For announcements (Schedules, Results, Admit Cards), do not fabricate vacancy or application dates
		_, _, examDate = ExtractDates(fullText, notice.PublishedAt)
	}

	// 11. Advt Number
	advtNo := notice.AdvtNumber
	if advtNo == "" {
		advtNo = ExtractAdvtNumber(fullText)
	}

	// 12. Official Government Portal URL
	officialURL := ExtractOfficialGovURL(notice.ContentText+" "+notice.NoticeTitle, notice.NoticeURL)

	// 13. Rich Summary Points
	summaryPoints := GenerateSummaryPoints(orgName, cleanTitle, noticeType, vacancies, qualification, appEnd, state, officialURL)

	// 14. Deterministic Confidence Score
	confidence := CalculateConfidenceScore(cleanTitle, orgName, noticeType, vacancies, qualification, appEnd, officialURL)

	return &job.ExtractedJobDetails{
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
		ExamDate:          examDate,
		FeeGeneral:        feeGen,
		FeeReserved:       feeRes,
		SalaryPayScale:    salary,
		OfficialApplyURL:  officialURL,
		OfficialNoticeURL: officialURL,
		SummaryPoints:     summaryPoints,
		ConfidenceScore:   confidence,
	}, nil
}

// CleanJobTitle strips HTML entities, excessive punctuation, and whitespace
func CleanJobTitle(title string) string {
	title = strings.ReplaceAll(title, "&#8211;", "–")
	title = strings.ReplaceAll(title, "&#8217;", "'")
	title = strings.ReplaceAll(title, "&#8216;", "'")
	title = strings.ReplaceAll(title, "&#8220;", "\"")
	title = strings.ReplaceAll(title, "&#8221;", "\"")
	title = strings.ReplaceAll(title, "&amp;", "&")
	title = strings.ReplaceAll(title, "&quot;", "\"")
	title = strings.ReplaceAll(title, "&nbsp;", " ")

	// Normalize spaces and line breaks
	spaceRegex := regexp.MustCompile(`\s+`)
	title = spaceRegex.ReplaceAllString(title, " ")
	title = strings.TrimSpace(title)

	// Remove trailing hyphens, colons or dots
	title = strings.TrimRight(title, " :-–.")
	return title
}

// ExtractNoticeType deterministically classifies the notice with word-boundary regexes
func ExtractNoticeType(text string) string {
	lower := strings.ToLower(text)

	// 1. Answer Key check
	answerKeyRegex := regexp.MustCompile(`(?i)\b(answer\s*keys?|response\s*sheets?|question\s*papers?|objection\s*trackers?|challenges?\s*thereon)\b`)
	if answerKeyRegex.MatchString(lower) {
		return "answer_key"
	}

	// 2. Admit Card / City Intimation / Hall Ticket check
	admitCardRegex := regexp.MustCompile(`(?i)\b(admit\s*cards?|hall\s*tickets?|call\s*letters?|e-admit|admission\s*certificates?|city\s*intimation|city\s*of\s*examination)\b`)
	if admitCardRegex.MatchString(lower) {
		return "admit_card"
	}

	// 3. Result / Merit List / Cut-off / Shortlisting check
	resultRegex := regexp.MustCompile(`(?i)\b(declaration\s*of\s*results?|final\s*results?|results?|marks\s*of|merit\s*lists?|score\s*cards?|cut-?offs?|shortlist(?:ed|ing)?|selection\s*lists?|candidature\s*cancelled|status\s*of\s*(?:shortlisted\s*)?candidates|shortlisting\s*of\s*candidates)\b`)
	if resultRegex.MatchString(lower) {
		return "result"
	}

	// 4. Interview Schedule / Exam Schedule / Date announcement check
	scheduleRegex := regexp.MustCompile(`(?i)\b(schedule\s+of|examination\s+schedule|exam\s+schedule|interview\s+schedule|schedule\s+for|tentative\s+schedule|revised\s+dates?|dates?\s+of\s+exam|time\s*table|schedule\s+of\s+pet|schedule\s+of\s+cbe)\b`)
	if scheduleRegex.MatchString(lower) {
		return "interview_schedule"
	}

	// 5. Corrigendum / Cancellation / Advisory
	corrigendumRegex := regexp.MustCompile(`(?i)\b(corrigendum|addendum|extension\s*of\s*dates?|cancellation|advisory|debarred)\b`)
	if corrigendumRegex.MatchString(lower) {
		return "corrigendum"
	}

	// Default is recruitment / vacancy announcement
	return "recruitment"
}

// ExtractRealOrg extracts authentic organization name using portal metadata and exact word boundary matches
func ExtractRealOrg(title, content, defaultPortal string) string {
	combined := title + " " + content

	// Known official portals
	portalMap := map[string]string{
		"UPSC":  "Union Public Service Commission (UPSC)",
		"SSC":   "Staff Selection Commission (SSC)",
		"IBPS":  "Institute of Banking Personnel Selection (IBPS)",
		"BPSC":  "Bihar Public Service Commission (BPSC)",
		"DSSSB": "Delhi Subordinate Services Selection Board (DSSSB)",
		"NTA":   "National Testing Agency (NTA)",
		"RRB":   "Railway Recruitment Board (RRB)",
		"ISRO":  "Indian Space Research Organisation (ISRO)",
		"UPPSC": "Uttar Pradesh Public Service Commission (UPPSC)",
	}

	// Check if title mentions specific departments within police or CAPFs
	if regexp.MustCompile(`(?i)\bdelhi\s*police\b`).MatchString(combined) {
		if defaultPortal == "SSC" {
			return "Staff Selection Commission (SSC) - Delhi Police"
		}
		return "Delhi Police"
	}
	if regexp.MustCompile(`(?i)\b(central\s*armed\s*police\s*forces|capfs?)\b`).MatchString(combined) {
		if defaultPortal == "SSC" {
			return "Staff Selection Commission (SSC) - CAPFs"
		}
		return "Central Armed Police Forces (CAPFs)"
	}
	if regexp.MustCompile(`(?i)\bassam\s*rifles\b`).MatchString(combined) {
		return "Assam Rifles"
	}

	// Check for standard recruiting organizations
	if org, ok := portalMap[strings.ToUpper(strings.TrimSpace(defaultPortal))]; ok {
		return org
	}

	// Check content pattern like "Indian Space Research Organisation has released..."
	orgRegex := regexp.MustCompile(`^([^()]+(?:\([^)]+\))?)\s+(?:has\s+released|has\s+published|has\s+invited|is\s+conducting)`)
	if match := orgRegex.FindStringSubmatch(content); len(match) > 1 {
		return strings.TrimSpace(match[1])
	}

	if defaultPortal != "" {
		return defaultPortal
	}
	return "Government of India"
}

// ExtractCategory categorizes the notice accurately using word-boundary checks
func ExtractCategory(sourcePortal, orgName, text string) string {
	combined := strings.ToLower(sourcePortal + " " + orgName + " " + text)

	if regexp.MustCompile(`(?i)\b(bank|banking|ibps|sbi|rbi|nabard|sebi|pnb|canara\s*bank|bank\s*of\s*baroda|union\s*bank)\b`).MatchString(combined) {
		return "Banking"
	}
	if regexp.MustCompile(`(?i)\b(railway|rrb|rrc|irctc|metro\s*rail|konkan\s*railway)\b`).MatchString(combined) {
		return "Railway"
	}
	if regexp.MustCompile(`(?i)\b(police|defence|defense|army|navy|air\s*force|crpf|bsf|cisf|itbp|assam\s*rifles|ssf|constable|sub-?inspector|rifleman|capfs?|nda|cds|military)\b`).MatchString(combined) {
		return "Defence & Police"
	}
	if regexp.MustCompile(`(?i)\b(teaching|teacher|lecturer|professor|faculty|ctet|ugc\s*net|b\.ed|prt|tgt|pgt|headmaster|principal)\b`).MatchString(combined) && !strings.Contains(combined, "recognized university") {
		return "Teaching"
	}
	if regexp.MustCompile(`(?i)\b(isro|drdo|bhel|ntpc|ongc|iocl|sail|bel|gate|engineering|engineer|junior\s*engineer|scientist)\b`).MatchString(combined) {
		return "Engineering & PSU"
	}
	if regexp.MustCompile(`(?i)\b(hospital|aiims|doctor|medical|nurse|nursing|pharmacist|mbbs|nhm|health\s*department)\b`).MatchString(combined) {
		return "Medical & Healthcare"
	}
	if regexp.MustCompile(`(?i)\b(court|judiciary|judge|law\s*officer|advocate|high\s*court|supreme\s*court|district\s*court)\b`).MatchString(combined) {
		return "Judiciary"
	}
	if regexp.MustCompile(`(?i)\b(state\s*govt|bpsc|uppsc|mppsc|rpsc|cgpsc|wbpsc|appsc|hpsc|jpsc|kpsc|opsc|ppsc|dsssb)\b`).MatchString(combined) {
		return "State Govt"
	}
	if regexp.MustCompile(`(?i)\b(central\s*govt|ssc|upsc|cgl|chsl|mts)\b`).MatchString(combined) {
		return "Central Govt"
	}

	return "Central Govt"
}

// ExtractIndianState identifies the target state or returns "All India"
func ExtractIndianState(title, content, org, category string) string {
	combined := strings.ToLower(title + " " + content + " " + org)

	stateMap := map[string][]string{
		"Andhra Pradesh":    {"andhra pradesh", "amaravati", "visakhapatnam", "vijayawada", "appsc"},
		"Arunachal Pradesh": {"arunachal pradesh", "itanagar"},
		"Assam":             {"assam", "guwahati", "dispur", "apsc"},
		"Bihar":             {"bihar", "patna", "bpsc", "bssc", "muzaffarpur"},
		"Chhattisgarh":      {"chhattisgarh", "raipur", "bilaspur", "cgpsc"},
		"Delhi":             {"delhi", "new delhi", "dsssb"},
		"Goa":               {"goa", "panaji", "gpsc goa"},
		"Gujarat":           {"gujarat", "gandhinagar", "ahmedabad", "gpsc"},
		"Haryana":           {"haryana", "chandigarh", "hpsc", "hssc"},
		"Himachal Pradesh":  {"himachal pradesh", "shimla", "hppsc"},
		"Jharkhand":         {"jharkhand", "ranchi", "jpsc", "jssc"},
		"Karnataka":         {"karnataka", "bengaluru", "bangalore", "kpsc"},
		"Kerala":            {"kerala", "thiruvananthapuram", "kochi"},
		"Madhya Pradesh":    {"madhya pradesh", "bhopal", "indore", "mppsc"},
		"Maharashtra":       {"maharashtra", "mumbai", "pune", "mpsc"},
		"Manipur":           {"manipur", "imphal"},
		"Meghalaya":         {"meghalaya", "shillong"},
		"Mizoram":           {"mizoram", "aizawl"},
		"Nagaland":          {"nagaland", "kohima"},
		"Odisha":            {"odisha", "bhubaneswar", "opsc", "osssc"},
		"Punjab":            {"punjab", "ppsc", "psssb"},
		"Rajasthan":         {"rajasthan", "jaipur", "rpsc", "rsmssb"},
		"Sikkim":            {"sikkim", "gangtok"},
		"Tamil Nadu":        {"tamil nadu", "chennai", "tnpsc"},
		"Telangana":         {"telangana", "hyderabad", "tspsc"},
		"Tripura":           {"tripura", "agartala"},
		"Uttar Pradesh":     {"uttar pradesh", "lucknow", "kanpur", "prayagraj", "uppsc", "upsssc"},
		"Uttarakhand":       {"uttarakhand", "dehradun", "ukpsc"},
		"West Bengal":       {"west bengal", "kolkata", "wbpsc"},
		"Jammu & Kashmir":   {"jammu", "kashmir", "srinagar", "jkssb"},
	}

	// For national recruitment commissions (SSC, UPSC, RRB, IBPS), the default is All India
	isNationalExam := strings.Contains(strings.ToLower(org), "staff selection commission") ||
		strings.Contains(strings.ToLower(org), "union public service commission") ||
		strings.Contains(strings.ToLower(org), "railway recruitment") ||
		strings.Contains(strings.ToLower(org), "institute of banking")

	if isNationalExam && !strings.Contains(combined, "only for state of") {
		return "All India"
	}

	for stateName, keywords := range stateMap {
		for _, kw := range keywords {
			pattern := `\b` + regexp.QuoteMeta(kw) + `\b`
			if regexp.MustCompile(`(?i)` + pattern).MatchString(combined) {
				return stateName
			}
		}
	}

	return "All India"
}

// ExtractVacancies extracts vacancy counts safely without confusing years, advt numbers or exam codes
func ExtractVacancies(title, content string) int {
	combined := title + " " + content

	// Pattern 1: e.g. "Vacancies: 14582", "Total Posts: 303", "11558 Posts", "4455 Vacancies"
	re1 := regexp.MustCompile(`(?i)\b(?:total\s*)?(?:vacanc(?:y|ies)|posts?|seats?)\b[:\s–-]*\b(\d{1,6})\b`)
	if match := re1.FindStringSubmatch(combined); len(match) > 1 {
		if v, err := strconv.Atoi(match[1]); err == nil && isValidVacancyCount(v) {
			return v
		}
	}

	re2 := regexp.MustCompile(`(?i)\b(\d{1,6})\b\s*(?:total\s*)?(?:posts?|vacanc(?:y|ies)|seats?)\b`)
	if match := re2.FindStringSubmatch(combined); len(match) > 1 {
		if v, err := strconv.Atoi(match[1]); err == nil && isValidVacancyCount(v) {
			return v
		}
	}

	// Pattern 3: "for 70 Driver posts"
	re3 := regexp.MustCompile(`(?i)\b(?:for|of)\s+(\d{1,6})\s+(?:various\s+)?(?:[a-zA-Z\s-]{1,25})\s*(?:posts?|vacanc(?:y|ies))\b`)
	if match := re3.FindStringSubmatch(combined); len(match) > 1 {
		if v, err := strconv.Atoi(match[1]); err == nil && isValidVacancyCount(v) {
			return v
		}
	}

	return 0
}

func isValidVacancyCount(v int) bool {
	if v <= 0 || v > 500000 {
		return false
	}
	// Avoid matching years like 2024, 2025, 2026, 2027 unless plausible
	if v >= 2020 && v <= 2030 {
		return false
	}
	return true
}

// ExtractAgeLimits extracts explicit age limit ranges or returns nil, nil if unavailable
func ExtractAgeLimits(text string) (*int, *int) {
	// Range: "between 18 and 32 years", "18 to 32 years", "Age Limit: 18-30 Years"
	rangeRegex := regexp.MustCompile(`(?i)\b(?:age(?:\s*limit)?|between)\b[:\s–-]*\b(\d{2})\b\s*(?:to|-|and)\s*\b(\d{2})\b\s*(?:years?|yrs?)?\b`)
	if match := rangeRegex.FindStringSubmatch(text); len(match) > 2 {
		minA, err1 := strconv.Atoi(match[1])
		maxA, err2 := strconv.Atoi(match[2])
		if err1 == nil && err2 == nil && minA >= 16 && maxA <= 65 && minA <= maxA {
			return &minA, &maxA
		}
	}

	var minAge *int
	var maxAge *int

	// Single min age: "Minimum Age: 18"
	minRegex := regexp.MustCompile(`(?i)\b(?:minimum\s*age|min\s*age|lower\s*age\s*limit)\b[:\s–-]*\b(\d{2})\b`)
	if match := minRegex.FindStringSubmatch(text); len(match) > 1 {
		if val, err := strconv.Atoi(match[1]); err == nil && val >= 16 && val <= 60 {
			minAge = &val
		}
	}

	// Single max age: "Maximum Age: 30", "Upper Age: 32", "Not exceeding 30 years"
	maxRegex := regexp.MustCompile(`(?i)\b(?:maximum\s*age|max\s*age|upper\s*age\s*limit|not\s*exceeding)\b[:\s–-]*\b(\d{2})\b`)
	if match := maxRegex.FindStringSubmatch(text); len(match) > 1 {
		if val, err := strconv.Atoi(match[1]); err == nil && val >= 18 && val <= 65 {
			maxAge = &val
		}
	}

	if minAge != nil || maxAge != nil {
		return minAge, maxAge
	}

	// Never return fabricated default values
	return nil, nil
}

// ExtractQualification identifies explicit educational requirements
func ExtractQualification(content, title string) string {
	combined := title + " " + content

	// Explicit label: "Qualification: Graduate in any discipline"
	labelRegex := regexp.MustCompile(`(?i)\b(?:minimum\s*|essential\s*|educational\s*)?qualification\b[:\s–-]*([^|\n\r;\.]+)`)
	if match := labelRegex.FindStringSubmatch(content); len(match) > 1 {
		q := strings.TrimSpace(match[1])
		if q != "" && q != "–" && q != "-" && len(q) > 3 {
			lowerQ := strings.ToLower(q)
			if strings.Contains(lowerQ, "10th") || strings.Contains(lowerQ, "matric") {
				return "10th Pass"
			}
			if strings.Contains(lowerQ, "12th") || strings.Contains(lowerQ, "intermediate") {
				return "12th Pass"
			}
			return q
		}
	}

	lower := strings.ToLower(combined)
	var quals []string

	if regexp.MustCompile(`(?i)\b(10th\s*pass|matriculation|matric|sslc)\b`).MatchString(lower) {
		quals = append(quals, "10th Pass")
	}
	if regexp.MustCompile(`(?i)\b(12th\s*pass|10\+2|intermediate|higher\s*secondary)\b`).MatchString(lower) {
		quals = append(quals, "12th Pass")
	}
	if regexp.MustCompile(`(?i)\b(iti\b|industrial\s*training\s*institute)\b`).MatchString(lower) {
		quals = append(quals, "ITI")
	}
	if regexp.MustCompile(`(?i)\b(diploma|polytechnic)\b`).MatchString(lower) {
		quals = append(quals, "Diploma")
	}
	if regexp.MustCompile(`(?i)\b(b\.e\.?|b\.tech\.?|engineering\s*degree)\b`).MatchString(lower) {
		quals = append(quals, "B.E / B.Tech")
	}
	if regexp.MustCompile(`(?i)\b(graduate|bachelor'?s?\s*degree|graduation|b\.a\.?|b\.sc\.?|b\.com\.?)\b`).MatchString(lower) {
		quals = append(quals, "Graduate")
	}
	if regexp.MustCompile(`(?i)\b(post\s*graduate|master'?s?\s*degree|m\.tech\.?|mba|m\.a\.?|m\.sc\.?)\b`).MatchString(lower) {
		quals = append(quals, "Post Graduate")
	}
	if regexp.MustCompile(`(?i)\b(mbbs|md/ms|bds|medical\s*degree)\b`).MatchString(lower) {
		quals = append(quals, "MBBS / Medical Degree")
	}
	if regexp.MustCompile(`(?i)\b(b\.ed\.?|d\.el\.ed\.?)\b`).MatchString(lower) {
		quals = append(quals, "B.Ed / D.El.Ed")
	}
	if regexp.MustCompile(`(?i)\b(ca|chartered\s*accountant|icwa)\b`).MatchString(lower) {
		quals = append(quals, "CA / Finance")
	}

	if len(quals) > 0 {
		return strings.Join(quals, ", ")
	}

	return "Refer to official notification"
}

// ExtractDates parses application start, end, and examination dates supporting multiple Indian formats
func ExtractDates(text, publishedAt string) (appStart, appEnd, examDate string) {
	// 1. Last date patterns (supporting filler phrases like "for application is", "is", "upto", etc.)
	lastDateRegexes := []*regexp.Regexp{
		regexp.MustCompile(`(?i)\b(?:last\s*date|closing\s*date|apply\s*till|end\s*date|reach\s*on\s*or\s*before)\b(?:\s+(?:for\s+application|is|of\s+submission|to\s+apply))*[:\s–-]*([0-3]?\d\s+(?:January|February|March|April|May|June|July|August|September|October|November|December|Jan|Feb|Mar|Apr|May|Jun|Jul|Aug|Sep|Oct|Nov|Dec)[a-z]*\s+\d{4})`),
		regexp.MustCompile(`(?i)\b(?:last\s*date|closing\s*date|apply\s*till|end\s*date|reach\s*on\s*or\s*before)\b(?:\s+(?:for\s+application|is|of\s+submission|to\s+apply))*[:\s–-]*([0-3]?\d[-/.][0-1]?\d[-/.]\d{2,4})`),
		regexp.MustCompile(`(?i)\b(?:till|before)\s+([0-3]?\d\s+(?:January|February|March|April|May|June|July|August|September|October|November|December|Jan|Feb|Mar|Apr|May|Jun|Jul|Aug|Sep|Oct|Nov|Dec)[a-z]*\s+\d{4})`),
	}

	for _, re := range lastDateRegexes {
		if match := re.FindStringSubmatch(text); len(match) > 1 {
			appEnd = strings.TrimSpace(match[1])
			break
		}
	}
	if appEnd == "" {
		appEnd = "Refer notification"
	}

	// 2. Start date patterns
	startDateRegexes := []*regexp.Regexp{
		regexp.MustCompile(`(?i)\b(?:start(?:ing)?\s*date|apply\s*online\s*from|opening\s*date|commencement\s*date)\b(?:\s+(?:for\s+application|is))*[:\s–-]*([0-3]?\d\s+(?:January|February|March|April|May|June|July|August|September|October|November|December|Jan|Feb|Mar|Apr|May|Jun|Jul|Aug|Sep|Oct|Nov|Dec)[a-z]*\s+\d{4})`),
		regexp.MustCompile(`(?i)\b(?:start(?:ing)?\s*date|apply\s*online\s*from|opening\s*date|commencement\s*date)\b(?:\s+(?:for\s+application|is))*[:\s–-]*([0-3]?\d[-/.][0-1]?\d[-/.]\d{2,4})`),
	}

	for _, re := range startDateRegexes {
		if match := re.FindStringSubmatch(text); len(match) > 1 {
			appStart = strings.TrimSpace(match[1])
			break
		}
	}
	if appStart == "" {
		if publishedAt != "" {
			appStart = publishedAt
		} else {
			appStart = "Refer notification"
		}
	}

	// 3. Exam date patterns
	examDateRegexes := []*regexp.Regexp{
		regexp.MustCompile(`(?i)\b(?:exam(?:ination)?\s*dates?|interview\s*(?:on|date)|cbe\s*dates?|paper-[iI]+\s*dates?)\b[:\s–-]*((?:[0-3]?\d\s+)?(?:January|February|March|April|May|June|July|August|September|October|November|December|Jan|Feb|Mar|Apr|May|Jun|Jul|Aug|Sep|Oct|Nov|Dec)[a-z]*(?:\s*(?:to|-)\s*(?:[a-zA-Z]+|\d{4}))?\s+\d{4})`),
		regexp.MustCompile(`(?i)\b(?:exam(?:ination)?\s*dates?|interview\s*(?:on|date)|cbe\s*dates?|paper-[iI]+\s*dates?)\b[:\s–-]*([0-3]?\d[-/.][0-1]?\d[-/.]\d{2,4})`),
		regexp.MustCompile(`(?i)\b(?:conducted\s*(?:on|in)|scheduled\s*(?:on|in|from))\s+((?:[0-3]?\d\s+)?(?:January|February|March|April|May|June|July|August|September|October|November|December|Jan|Feb|Mar|Apr|May|Jun|Jul|Aug|Sep|Oct|Nov|Dec)[a-z]*(?:\s*-\s*[a-zA-Z]+)?\s+\d{4})`),
	}

	for _, re := range examDateRegexes {
		if match := re.FindStringSubmatch(text); len(match) > 1 {
			examDate = strings.TrimSpace(match[1])
			break
		}
	}
	if examDate == "" {
		examDate = "To be announced"
	}

	return appStart, appEnd, examDate
}

// ExtractFees extracts general and reserved category fees
func ExtractFees(text string) (string, string) {
	lower := strings.ToLower(text)

	feeGen := "Refer official notification"
	feeRes := "Refer official notification"

	// Check explicit numeric fee first (e.g. ₹100, Rs. 500, Fee: 100)
	feeRegex := regexp.MustCompile(`(?i)\b(?:application\s*)?fee\b[:\s–-]*(?:rs\.?|₹|\$)?\s*(\d{1,5})\b`)
	if match := feeRegex.FindStringSubmatch(text); len(match) > 1 {
		feeGen = fmt.Sprintf("₹%s", match[1])
	}

	// If no fee amount was matched, check if all applications are free
	if feeGen == "Refer official notification" {
		if regexp.MustCompile(`(?i)\b(no\s*application\s*fee|no\s*fee|free\s*of\s*cost|nil\s*application\s*fee)\b`).MatchString(lower) {
			return "Free / Nil", "Free / Nil"
		}
	}

	// Check reserved category exemption or fee
	if regexp.MustCompile(`(?i)\b(?:exempted|nil|free|₹0)\s*(?:for\s*(?:sc|st|women|female|pwd|pwbd|esm)|to\s*all)\b|\bfree\s*for\s*(?:sc|st|women|female)\b|\b(?:sc|st|women|female|pwd|pwbd|esm)[a-z\s,/]*\bare\s*exempted\b`).MatchString(lower) {
		feeRes = "₹0 for SC/ST/Women/PwBD"
	} else if match := regexp.MustCompile(`(?i)\b(?:fee\s*for\s*reserved|fee\s*reserved|sc/st\s*fee)\b[:\s–-]*(?:rs\.?|₹|\$)?\s*(\d{1,5})\b`).FindStringSubmatch(text); len(match) > 1 {
		feeRes = fmt.Sprintf("₹%s", match[1])
	} else if feeGen != "Refer official notification" {
		feeRes = "As per rules"
	}

	return feeGen, feeRes
}

// ExtractSalary extracts pay matrix level, scale, or monthly emoluments
func ExtractSalary(text string) string {
	// Pay Level pattern: Level 4 to Level 8, Level-7, Pay Matrix Level 10
	levelRegex := regexp.MustCompile(`(?i)\b(?:pay\s*matrix\s*)?(level[- ]\d+(?:\s*(?:to|-)\s*(?:level[- ]?)?\d+)?)\b`)
	if match := levelRegex.FindStringSubmatch(text); len(match) > 1 {
		return strings.ToUpper(strings.TrimSpace(match[1]))
	}

	// Salary range: Rs. 25,500 - 81,100
	rangeRegex := regexp.MustCompile(`(?i)\b(?:salary\s*[:–-]|pay\s*scale\s*[:–-]|scale\s*of\s*pay\s*[:–-]|monthly\s*salary\s*of\s*)\s*(?:rs\.?|₹)?\s*([\d,]+(?:\s*(?:to|-)\s*[\d,]+)?(?:\s*(?:per\s*month|\/-|p\.m\.))?)`)
	if match := rangeRegex.FindStringSubmatch(text); len(match) > 1 {
		return "₹" + strings.TrimPrefix(strings.TrimSpace(match[1]), "Rs. ")
	}

	// Grade Pay
	gpRegex := regexp.MustCompile(`(?i)\b(grade\s*pay\s*(?:rs\.?|₹)?\s*\d+)\b`)
	if match := gpRegex.FindStringSubmatch(text); len(match) > 1 {
		return strings.ToUpper(strings.TrimSpace(match[1]))
	}

	return ""
}

// ExtractAdvtNumber extracts notification or advertisement reference numbers
func ExtractAdvtNumber(text string) string {
	advtRegex := regexp.MustCompile(`(?i)\b(?:advt\.?\s*no\.?|notification\s*no\.?|cen\s*no\.?|notice\s*no\.?)\b[:\s–-]*([a-zA-Z0-9/._-]+)`)
	if match := advtRegex.FindStringSubmatch(text); len(match) > 1 {
		res := strings.TrimSpace(match[1])
		if len(res) >= 3 && !strings.EqualFold(res, "nil") {
			return res
		}
	}
	return ""
}

// ExtractOfficialGovURL validates and extracts real official government/academic domains
func ExtractOfficialGovURL(text string, fallbackURL string) string {
	lowerFallback := strings.ToLower(fallbackURL)

	// Exclude aggregator domains
	if strings.Contains(lowerFallback, "freejobalert.com") ||
		strings.Contains(lowerFallback, "sarkariresult") ||
		strings.Contains(lowerFallback, "blogspot") ||
		strings.Contains(lowerFallback, "wordpress") {
		return ""
	}

	// Accept verified official domains
	if strings.Contains(lowerFallback, ".gov.in") ||
		strings.Contains(lowerFallback, ".nic.in") ||
		strings.Contains(lowerFallback, ".ac.in") ||
		strings.Contains(lowerFallback, ".edu.in") ||
		strings.Contains(lowerFallback, "ibps.in") {
		return fallbackURL
	}

	govRegex := regexp.MustCompile(`(?i)(https?://[a-zA-Z0-9.-]+\.(?:gov\.in|nic\.in|ac\.in|edu\.in|org\.in|res\.in|mil\.in)(?:/[^\s"<>]*)?)`)
	if match := govRegex.FindStringSubmatch(text); len(match) > 1 {
		raw := match[1]
		if u, err := url.Parse(raw); err == nil && u.Host != "" {
			return raw
		}
	}

	return fallbackURL
}

// GenerateSummaryPoints builds structured highlight bullets based solely on verified fields
func GenerateSummaryPoints(org, title, noticeType string, vacancies int, qual, lastDate, state, officialURL string) []string {
	var points []string

	if vacancies > 0 {
		points = append(points, fmt.Sprintf("%d Total Vacancies announced by %s", vacancies, org))
	} else {
		displayType := strings.ReplaceAll(noticeType, "_", " ")
		points = append(points, fmt.Sprintf("Official %s update announced by %s", displayType, org))
	}

	if qual != "Refer to official notification" && qual != "" {
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

// CalculateConfidenceScore deterministically scores extraction completeness
func CalculateConfidenceScore(title, org, noticeType string, vacancies int, qual, lastDate, officialURL string) float64 {
	score := 0.70

	if title != "" {
		score += 0.05
	}
	if org != "" && org != "Government of India" {
		score += 0.05
	}
	if noticeType != "" {
		score += 0.05
	}
	if vacancies > 0 {
		score += 0.05
	}
	if qual != "Refer to official notification" && qual != "" {
		score += 0.05
	}
	if lastDate != "Refer notification" && lastDate != "" {
		score += 0.05
	}
	if officialURL != "" {
		score += 0.05
	}

	if score > 1.0 {
		score = 1.0
	}
	return score
}
