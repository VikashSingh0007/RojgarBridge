package models

import (
	"time"
)

type JobStatus string

const (
	StatusDraft     JobStatus = "draft"
	StatusPublished JobStatus = "published"
	StatusRejected  JobStatus = "rejected"
)

// RawNotice represents raw data picked up by the web scraper
type RawNotice struct {
	SourcePortal string    `json:"source_portal"` // e.g., "UPSC", "SSC", "IBPS"
	NoticeTitle  string    `json:"notice_title"`  // Original raw title
	NoticeURL    string    `json:"notice_url"`    // Page or PDF URL
	AdvtNumber   string    `json:"advt_number"`   // Advertisement/Notification number
	PublishedAt  string    `json:"published_at"`  // Date string as on portal
	FetchedAt    time.Time `json:"fetched_at"`
	ContentText  string    `json:"content_text"`  // Raw snippet or extracted text
}

// ExtractedJobDetails represents the AI-parsed structured details
type ExtractedJobDetails struct {
	Title             string   `json:"title"`
	NoticeType        string   `json:"notice_type"` // "recruitment", "result", "admit_card", "interview"
	Organization      string   `json:"organization"`
	Category          string   `json:"category"` // Central, State, Banking, Defence, Railway, etc.
	State             string   `json:"state"`    // "All India" or specific state
	AdvtNumber        string   `json:"advt_number"`
	TotalVacancies    int      `json:"total_vacancies"`
	MinAge            *int     `json:"min_age,omitempty"`
	MaxAge            *int     `json:"max_age,omitempty"`
	Qualification     string   `json:"qualification"`
	ApplicationStart  string   `json:"application_start"` // YYYY-MM-DD or raw date string
	ApplicationEnd    string   `json:"application_end"`   // YYYY-MM-DD or raw date string
	ExamDate          string   `json:"exam_date"`
	FeeGeneral        string   `json:"fee_general"`
	FeeReserved       string   `json:"fee_reserved"`
	SalaryPayScale    string   `json:"salary_payscale"`
	OfficialApplyURL  string   `json:"official_apply_url"`
	OfficialNoticeURL string   `json:"official_notice_url"`
	SummaryPoints     []string `json:"summary_points"`
	ConfidenceScore   float64  `json:"confidence_score"` // 0.0 - 1.0 (AI confidence)
}

// Job represents the database record
type Job struct {
	ID           int64               `json:"id"`
	Hash         string              `json:"hash"` // Unique deduplication hash
	SourcePortal string              `json:"source_portal"`
	Status       JobStatus           `json:"status"` // draft, published, rejected
	RawNotice    RawNotice           `json:"raw_notice"`
	Details      ExtractedJobDetails `json:"details"`
	ReviewNotes  string              `json:"review_notes,omitempty"`
	CreatedAt    time.Time           `json:"created_at"`
	UpdatedAt    time.Time           `json:"updated_at"`
	PublishedAt  *time.Time          `json:"published_at,omitempty"`
}
