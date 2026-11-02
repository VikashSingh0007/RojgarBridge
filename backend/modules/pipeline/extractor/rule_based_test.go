package extractor_test

import (
	"testing"
	"time"

	"rojgarbridge-backend/modules/job"
	"rojgarbridge-backend/modules/pipeline/extractor"
)

func TestCleanJobTitle(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{
			input:    "Notice &#8211; Recruitment of Assistant Manager &amp; Engineers: ",
			expected: "Notice – Recruitment of Assistant Manager & Engineers",
		},
		{
			input:    "Candidate&#8217;s Response Sheet&#8220;Final&#8221;   \n",
			expected: "Candidate's Response Sheet\"Final\"",
		},
	}

	for _, tc := range tests {
		actual := extractor.CleanJobTitle(tc.input)
		if actual != tc.expected {
			t.Errorf("CleanJobTitle(%q) = %q; want %q", tc.input, actual, tc.expected)
		}
	}
}

func TestExtractNoticeType(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{
			input:    "Declaration of result of PET/PST to shortlist candidates for appearing in Paper-II",
			expected: "result",
		},
		{
			input:    "Information regarding the city of examination and Admission Certificate for Junior Engineer",
			expected: "admit_card",
		},
		{
			input:    "Candidate's Response Sheet along with Tentative Answer Keys and inviting challenges",
			expected: "answer_key",
		},
		{
			input:    "Important Notice - Revised dates for Junior Engineer Examination, 2026",
			expected: "interview_schedule",
		},
		{
			input:    "Combined Graduate Level Examination 2026 - Notice of Vacancy",
			expected: "recruitment",
		},
	}

	for _, tc := range tests {
		actual := extractor.ExtractNoticeType(tc.input)
		if actual != tc.expected {
			t.Errorf("ExtractNoticeType(%q) = %q; want %q", tc.input, actual, tc.expected)
		}
	}
}

func TestExtractRealOrg(t *testing.T) {
	tests := []struct {
		title    string
		content  string
		portal   string
		expected string
	}{
		{
			title:    "Sub-Inspector in Delhi Police Examination 2025",
			content:  "Staff Selection Commission has announced Delhi Police vacancies",
			portal:   "SSC",
			expected: "Staff Selection Commission (SSC) - Delhi Police",
		},
		{
			title:    "Constable (GD) in Central Armed Police Forces (CAPFs) Examination 2026",
			content:  "CAPFs recruitment",
			portal:   "SSC",
			expected: "Staff Selection Commission (SSC) - CAPFs",
		},
		{
			title:    "Civil Services Examination 2026",
			content:  "Union Public Service Commission",
			portal:   "UPSC",
			expected: "Union Public Service Commission (UPSC)",
		},
		{
			title:    "Scientist 'SC' Recruitment",
			content:  "Careers at space agency",
			portal:   "ISRO",
			expected: "Indian Space Research Organisation (ISRO)",
		},
	}

	for _, tc := range tests {
		actual := extractor.ExtractRealOrg(tc.title, tc.content, tc.portal)
		if actual != tc.expected {
			t.Errorf("ExtractRealOrg(%q, %q) = %q; want %q", tc.title, tc.portal, actual, tc.expected)
		}
	}
}

func TestExtractCategory(t *testing.T) {
	tests := []struct {
		portal   string
		org      string
		text     string
		expected string
	}{
		{portal: "SSC", org: "Staff Selection Commission", text: "Sub-Inspector in Delhi Police and CAPFs", expected: "Defence & Police"},
		{portal: "SSC", org: "Staff Selection Commission", text: "Junior Engineer (Civil, Mechanical, Electrical)", expected: "Engineering & PSU"},
		{portal: "IBPS", org: "IBPS", text: "Probationary Officers in Participating Public Sector Banks", expected: "Banking"},
		{portal: "UPSC", org: "UPSC", text: "Assistant Professor in Central Universities", expected: "Teaching"},
		{portal: "SSC", org: "Staff Selection Commission", text: "Combined Graduate Level Examination (CGL)", expected: "Central Govt"},
	}

	for _, tc := range tests {
		actual := extractor.ExtractCategory(tc.portal, tc.org, tc.text)
		if actual != tc.expected {
			t.Errorf("ExtractCategory(%q, %q, %q) = %q; want %q", tc.portal, tc.org, tc.text, actual, tc.expected)
		}
	}
}

func TestExtractVacancies(t *testing.T) {
	tests := []struct {
		title    string
		content  string
		expected int
	}{
		{
			title:    "SSC CGL 2025 - 14582 Posts announced",
			content:  "Total Vacancies: 14582 across Ministries",
			expected: 14582,
		},
		{
			title:    "Tentative Vacancies of Junior Engineer Examination, 2026",
			content:  "Recruitment for 1765 posts of Junior Engineer",
			expected: 1765,
		},
		{
			title:    "Junior Engineer Examination, 2026 (Paper-II held in 2025)",
			content:  "Notice regarding examination dates 2026",
			expected: 0, // 2026 and 2025 are years, not vacancies
		},
	}

	for _, tc := range tests {
		actual := extractor.ExtractVacancies(tc.title, tc.content)
		if actual != tc.expected {
			t.Errorf("ExtractVacancies(%q, %q) = %d; want %d", tc.title, tc.content, actual, tc.expected)
		}
	}
}

func TestExtractAgeLimits(t *testing.T) {
	// Case 1: Explicit range
	textRange := "Candidates must be between 18 and 30 years as on crucial date."
	minA, maxA := extractor.ExtractAgeLimits(textRange)
	if minA == nil || maxA == nil || *minA != 18 || *maxA != 30 {
		t.Fatalf("Expected 18 and 30, got %v and %v", minA, maxA)
	}

	// Case 2: No age limit mentioned in text (must NOT return fabricated defaults)
	textNone := "Tentative Vacancies of Grade C Stenographers Examination 2025."
	minNone, maxNone := extractor.ExtractAgeLimits(textNone)
	if minNone != nil || maxNone != nil {
		t.Fatalf("Expected nil, nil for notice without age limits, got %v and %v", minNone, maxNone)
	}

	// Case 3: Upper age limit only
	textUpper := "Upper age limit is not exceeding 35 years."
	_, maxUpper := extractor.ExtractAgeLimits(textUpper)
	if maxUpper == nil || *maxUpper != 35 {
		t.Fatalf("Expected maxAge 35, got %v", maxUpper)
	}
}

func TestExtractQualification(t *testing.T) {
	tests := []struct {
		title    string
		content  string
		expected string
	}{
		{
			title:    "Junior Engineer Civil",
			content:  "Qualification: Diploma or Degree in Civil Engineering",
			expected: "Diploma or Degree in Civil Engineering",
		},
		{
			title:    "Combined Graduate Level 2026",
			content:  "Candidate must possess Bachelor's Degree / Graduate in any discipline",
			expected: "Graduate",
		},
		{
			title:    "Multi Tasking Staff Examination",
			content:  "Candidate must have passed 10th Pass / Matriculation examination",
			expected: "10th Pass",
		},
		{
			title:    "Corrigendum Notice",
			content:  "Notice regarding examination centre change",
			expected: "Refer to official notification",
		},
	}

	for _, tc := range tests {
		actual := extractor.ExtractQualification(tc.content, tc.title)
		if actual != tc.expected {
			t.Errorf("ExtractQualification(%q) = %q; want %q", tc.title, actual, tc.expected)
		}
	}
}

func TestExtractDates(t *testing.T) {
	text := "Apply online from 24-09-2026. Closing date for application is 24 November 2026. Exam Date: Dec 2026."
	start, end, exam := extractor.ExtractDates(text, "2026-09-24")

	if start != "24-09-2026" {
		t.Errorf("Expected start 24-09-2026, got %s", start)
	}
	if end != "24 November 2026" {
		t.Errorf("Expected end '24 November 2026', got %s", end)
	}
	if exam != "Dec 2026" {
		t.Errorf("Expected exam 'Dec 2026', got %s", exam)
	}
}

func TestExtractOfficialGovURL(t *testing.T) {
	// Official .gov.in URL
	validURL := "https://ssc.gov.in/api/attachment/writeup_29102026.pdf"
	if out := extractor.ExtractOfficialGovURL("", validURL); out != validURL {
		t.Errorf("Expected %s, got %s", validURL, out)
	}

	// Third party aggregator URL must be rejected
	aggregatorURL := "https://www.freejobalert.com/ssc-cgl/12345/"
	if out := extractor.ExtractOfficialGovURL("", aggregatorURL); out != "" {
		t.Errorf("Expected empty string for aggregator, got %s", out)
	}
}

func TestRuleBasedExtractorEndToEnd(t *testing.T) {
	ext := extractor.NewRuleBasedExtractor()

	notice := job.RawNotice{
		SourcePortal: "SSC",
		NoticeTitle:  "Constable (GD) in Central Armed Police Forces (CAPFs) Examination, 2026",
		NoticeURL:    "https://ssc.gov.in/api/attachment/writeup_23102026.pdf",
		PublishedAt:  "2026-10-23",
		FetchedAt:    time.Now(),
		ContentText:  "Title: Constable (GD) in Central Armed Police Forces (CAPFs) Examination, 2026 | Source: Staff Selection Commission (Official Portal: ssc.gov.in) | Total Vacancies: 39481 Posts | Qualification: 10th Pass | Age limit: 18 to 23 years | Application Fee: ₹100, Free for Women and SC/ST | Pay Scale: Level-3 (Rs. 21,700 - 69,100) | Last Date: 14 October 2026 | Official Document: https://ssc.gov.in/api/attachment/writeup_23102026.pdf",
	}

	details, err := ext.Extract(notice)
	if err != nil {
		t.Fatalf("Extract returned error: %v", err)
	}

	if details.Title != "Constable (GD) in Central Armed Police Forces (CAPFs) Examination, 2026" {
		t.Errorf("Unexpected title: %s", details.Title)
	}
	if details.NoticeType != "recruitment" {
		t.Errorf("Expected notice type recruitment, got %s", details.NoticeType)
	}
	if details.Category != "Defence & Police" {
		t.Errorf("Expected Category Defence & Police, got %s", details.Category)
	}
	if details.State != "All India" {
		t.Errorf("Expected State All India, got %s", details.State)
	}
	if details.TotalVacancies != 39481 {
		t.Errorf("Expected 39481 vacancies, got %d", details.TotalVacancies)
	}
	if details.MinAge == nil || *details.MinAge != 18 || details.MaxAge == nil || *details.MaxAge != 23 {
		t.Errorf("Expected age 18-23, got %v-%v", details.MinAge, details.MaxAge)
	}
	if details.Qualification != "10th Pass" {
		t.Errorf("Expected 10th Pass, got %s", details.Qualification)
	}
	if details.ApplicationEnd != "14 October 2026" {
		t.Errorf("Expected Last Date '14 October 2026', got %s", details.ApplicationEnd)
	}
	if details.FeeGeneral != "₹100" {
		t.Errorf("Expected Fee ₹100, got %s", details.FeeGeneral)
	}
	if details.FeeReserved != "₹0 for SC/ST/Women/PwBD" {
		t.Errorf("Expected reserved fee exemption, got %s", details.FeeReserved)
	}
	if details.SalaryPayScale != "LEVEL-3" {
		t.Errorf("Expected LEVEL-3 salary, got %s", details.SalaryPayScale)
	}
	if details.OfficialNoticeURL != "https://ssc.gov.in/api/attachment/writeup_23102026.pdf" {
		t.Errorf("Expected official URL, got %s", details.OfficialNoticeURL)
	}
	if len(details.SummaryPoints) == 0 {
		t.Errorf("Expected summary points, got empty")
	}
}

