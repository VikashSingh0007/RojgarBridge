package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"rojgarbridge-backend/internal/db"
	"rojgarbridge-backend/internal/models"
	"rojgarbridge-backend/internal/notifier"
	"rojgarbridge-backend/internal/scrapers"
)

type Handler struct {
	db       *db.DB
	manager  *scrapers.Manager
	notifier notifier.Notifier
}

func NewHandler(database *db.DB, manager *scrapers.Manager, notif notifier.Notifier) *Handler {
	return &Handler{
		db:       database,
		manager:  manager,
		notifier: notif,
	}
}

func respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

func respondError(w http.ResponseWriter, status int, message string) {
	respondJSON(w, status, map[string]string{"error": message})
}

// Health check
func (h *Handler) Health(w http.ResponseWriter, r *http.Request) {
	respondJSON(w, http.StatusOK, map[string]string{
		"status":  "healthy",
		"service": "RojgarBridge Official Government Job Engine",
	})
}

// parseLimit helper parses limit query parameter
func parseLimit(limitStr string, defaultLimit int) int {
	if strings.EqualFold(limitStr, "all") || limitStr == "-1" {
		return 0 // 0 means return all
	}
	if limitStr == "" {
		return defaultLimit
	}
	val, err := strconv.Atoi(limitStr)
	if err != nil || val <= 0 {
		return defaultLimit
	}
	return val
}

func getQuerySlice(r *http.Request, key string) []string {
	values := r.URL.Query()[key]
	var result []string
	for _, v := range values {
		parts := strings.Split(v, ",")
		for _, p := range parts {
			trimmed := strings.TrimSpace(p)
			if trimmed != "" && !strings.EqualFold(trimmed, "all") {
				result = append(result, trimmed)
			}
		}
	}
	return result
}

// GetAllJobs - lists all jobs (draft + published) with optional filters
func (h *Handler) GetAllJobs(w http.ResponseWriter, r *http.Request) {
	limit := parseLimit(r.URL.Query().Get("limit"), 100)
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	if offset < 0 {
		offset = 0
	}
	status := strings.TrimSpace(r.URL.Query().Get("status"))
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	sortBy := strings.TrimSpace(r.URL.Query().Get("sort"))
	categories := getQuerySlice(r, "category")
	qualifications := getQuerySlice(r, "qualification")
	states := getQuerySlice(r, "state")
	noticeTypes := getQuerySlice(r, "type")

	jobs, total, err := h.db.QueryJobs(db.JobFilterParams{
		Status:         status,
		Query:          q,
		NoticeTypes:    noticeTypes,
		Categories:     categories,
		Qualifications: qualifications,
		States:         states,
		SortBy:         sortBy,
		Limit:          limit,
		Offset:         offset,
	})
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respondJSON(w, http.StatusOK, map[string]interface{}{
		"total":  total,
		"count":  len(jobs),
		"limit":  limit,
		"offset": offset,
		"jobs":   jobs,
	})
}

// GetDrafts - lists all jobs awaiting review
func (h *Handler) GetDrafts(w http.ResponseWriter, r *http.Request) {
	limit := parseLimit(r.URL.Query().Get("limit"), 100)
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	if offset < 0 {
		offset = 0
	}
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	sortBy := strings.TrimSpace(r.URL.Query().Get("sort"))
	categories := getQuerySlice(r, "category")
	qualifications := getQuerySlice(r, "qualification")
	states := getQuerySlice(r, "state")
	noticeTypes := getQuerySlice(r, "type")

	jobs, total, err := h.db.QueryJobs(db.JobFilterParams{
		Status:         string(models.StatusDraft),
		Query:          q,
		NoticeTypes:    noticeTypes,
		Categories:     categories,
		Qualifications: qualifications,
		States:         states,
		SortBy:         sortBy,
		Limit:          limit,
		Offset:         offset,
	})
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respondJSON(w, http.StatusOK, map[string]interface{}{
		"total":  total,
		"count":  len(jobs),
		"limit":  limit,
		"offset": offset,
		"jobs":   jobs,
	})
}

// GetPublished - lists live jobs for public viewing with full DB filtering
func (h *Handler) GetPublished(w http.ResponseWriter, r *http.Request) {
	limit := parseLimit(r.URL.Query().Get("limit"), 10)
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	if offset < 0 {
		offset = 0
	}
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	sortBy := strings.TrimSpace(r.URL.Query().Get("sort"))
	categories := getQuerySlice(r, "category")
	qualifications := getQuerySlice(r, "qualification")
	states := getQuerySlice(r, "state")
	noticeTypes := getQuerySlice(r, "type")

	jobs, total, err := h.db.QueryJobs(db.JobFilterParams{
		Status:         string(models.StatusPublished),
		Query:          q,
		NoticeTypes:    noticeTypes,
		Categories:     categories,
		Qualifications: qualifications,
		States:         states,
		SortBy:         sortBy,
		Limit:          limit,
		Offset:         offset,
	})
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respondJSON(w, http.StatusOK, map[string]interface{}{
		"total":  total,
		"count":  len(jobs),
		"limit":  limit,
		"offset": offset,
		"jobs":   jobs,
	})
}

// SearchJobs - full text search with qualification, state, category, type and sort filters
func (h *Handler) SearchJobs(w http.ResponseWriter, r *http.Request) {
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	status := strings.TrimSpace(r.URL.Query().Get("status"))
	if status == "" {
		status = string(models.StatusPublished)
	}
	limit := parseLimit(r.URL.Query().Get("limit"), 10)
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	if offset < 0 {
		offset = 0
	}
	sortBy := strings.TrimSpace(r.URL.Query().Get("sort"))
	categories := getQuerySlice(r, "category")
	qualifications := getQuerySlice(r, "qualification")
	states := getQuerySlice(r, "state")
	noticeTypes := getQuerySlice(r, "type")

	jobs, total, err := h.db.QueryJobs(db.JobFilterParams{
		Status:         status,
		Query:          q,
		NoticeTypes:    noticeTypes,
		Categories:     categories,
		Qualifications: qualifications,
		States:         states,
		SortBy:         sortBy,
		Limit:          limit,
		Offset:         offset,
	})
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"query":  q,
		"total":  total,
		"count":  len(jobs),
		"limit":  limit,
		"offset": offset,
		"jobs":   jobs,
	})
}

// GetClosingSoon - returns jobs expiring in the upcoming days
func (h *Handler) GetClosingSoon(w http.ResponseWriter, r *http.Request) {
	days, _ := strconv.Atoi(r.URL.Query().Get("days"))
	if days <= 0 {
		days = 7
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit <= 0 {
		limit = 50
	}

	jobs, err := h.db.GetClosingSoonJobs(days, limit)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"days_window": days,
		"count":       len(jobs),
		"jobs":        jobs,
	})
}

// GetAnalytics - returns aggregated dashboard metrics
func (h *Handler) GetAnalytics(w http.ResponseWriter, r *http.Request) {
	analytics, err := h.db.GetAnalytics()
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respondJSON(w, http.StatusOK, analytics)
}

// GetJobByID - details of a single job
func (h *Handler) GetJobByID(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid job id")
		return
	}

	job, err := h.db.GetJobByID(id)
	if err != nil {
		respondError(w, http.StatusNotFound, "job not found")
		return
	}

	respondJSON(w, http.StatusOK, job)
}

// UpdateJob - edit job details prior to publishing
func (h *Handler) UpdateJob(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid job id")
		return
	}

	job, err := h.db.GetJobByID(id)
	if err != nil {
		respondError(w, http.StatusNotFound, "job not found")
		return
	}

	var payload struct {
		Details     models.ExtractedJobDetails `json:"details"`
		ReviewNotes string                     `json:"review_notes"`
	}

	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		respondError(w, http.StatusBadRequest, "malformed request payload: "+err.Error())
		return
	}

	job.Details = payload.Details
	if payload.ReviewNotes != "" {
		job.ReviewNotes = payload.ReviewNotes
	}

	if err := h.db.UpdateJob(job); err != nil {
		respondError(w, http.StatusInternalServerError, "failed to update job: "+err.Error())
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"message": "job updated successfully",
		"job":     job,
	})
}

// PublishJob - approve draft and publish
func (h *Handler) PublishJob(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid job id")
		return
	}

	job, err := h.db.GetJobByID(id)
	if err != nil {
		respondError(w, http.StatusNotFound, "job not found")
		return
	}

	if err := h.db.PublishJob(id); err != nil {
		respondError(w, http.StatusInternalServerError, "failed to publish job: "+err.Error())
		return
	}

	job.Status = models.StatusPublished

	// Broadcast notification to Telegram
	if h.notifier != nil {
		go func(j *models.Job) {
			_ = h.notifier.NotifyJobPublished(j)
		}(job)
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"message": "job published successfully",
		"id":      id,
		"status":  models.StatusPublished,
	})
}

// BulkPublish - publish all or selected drafts
func (h *Handler) BulkPublish(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		IDs []int64 `json:"ids"`
	}
	_ = json.NewDecoder(r.Body).Decode(&payload)

	count, err := h.db.BulkPublish(payload.IDs)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to bulk publish: "+err.Error())
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"message":         "Jobs published successfully",
		"published_count": count,
	})
}

// RejectJob - reject draft
func (h *Handler) RejectJob(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid job id")
		return
	}

	var payload struct {
		Reason string `json:"reason"`
	}
	_ = json.NewDecoder(r.Body).Decode(&payload)
	if payload.Reason == "" {
		payload.Reason = "Rejected by admin"
	}

	if err := h.db.RejectJob(id, payload.Reason); err != nil {
		respondError(w, http.StatusInternalServerError, "failed to reject job: "+err.Error())
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"message": "job rejected",
		"id":      id,
		"status":  models.StatusRejected,
	})
}

// TriggerPipeline - manual trigger to scrape and extract
func (h *Handler) TriggerPipeline(w http.ResponseWriter, r *http.Request) {
	go func() {
		h.manager.RunPipeline()
	}()

	respondJSON(w, http.StatusAccepted, map[string]string{
		"message": "Scraping and AI extraction pipeline triggered in background across official government portals",
	})
}

// ReExtractDrafts - re-runs improved extraction on existing drafts
func (h *Handler) ReExtractDrafts(w http.ResponseWriter, r *http.Request) {
	go func() {
		h.manager.ReExtractDrafts()
	}()

	respondJSON(w, http.StatusAccepted, map[string]string{
		"message": "Re-extraction started for existing draft jobs in background",
	})
}

// GetRSSFeed - serves standard RSS 2.0 XML for Google SEO
func (h *Handler) GetRSSFeed(w http.ResponseWriter, r *http.Request) {
	jobs, err := h.db.GetRecentPublished(50)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/rss+xml; charset=utf-8")
	fmt.Fprintf(w, `<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0" xmlns:atom="http://www.w3.org/2005/Atom">
<channel>
<title>RojgarHub - Official Sarkari Naukri &amp; Govt Job Alerts</title>
<link>https://rojgarhub.in/</link>
<description>Latest verified Indian government jobs, exam notices, and admit cards directly from official portals.</description>
<language>en-in</language>
<lastBuildDate>%s</lastBuildDate>
`, time.Now().Format(time.RFC1123Z))

	for _, job := range jobs {
		link := job.Details.OfficialApplyURL
		if link == "" {
			link = job.RawNotice.NoticeURL
		}
		fmt.Fprintf(w, `<item>
<title><![CDATA[%s]]></title>
<link>%s</link>
<guid isPermaLink="false">%s</guid>
<description><![CDATA[Organization: %s | Category: %s | Vacancies: %d | Eligibility: %s | Last Date: %s]]></description>
<pubDate>%s</pubDate>
</item>
`,
			job.Details.Title,
			link,
			job.Hash,
			job.Details.Organization,
			job.Details.Category,
			job.Details.TotalVacancies,
			job.Details.Qualification,
			job.Details.ApplicationEnd,
			job.CreatedAt.Format(time.RFC1123Z),
		)
	}

	fmt.Fprint(w, `</channel>
</rss>`)
}

// GetSitemap - serves XML sitemap for SEO crawlers
func (h *Handler) GetSitemap(w http.ResponseWriter, r *http.Request) {
	jobs, err := h.db.GetRecentPublished(200)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/xml; charset=utf-8")
	fmt.Fprint(w, `<?xml version="1.0" encoding="UTF-8"?>
<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
<url>
<loc>https://rojgarhub.in/</loc>
<changefreq>hourly</changefreq>
<priority>1.0</priority>
</url>
`)

	for _, job := range jobs {
		link := job.Details.OfficialApplyURL
		if link == "" {
			link = job.RawNotice.NoticeURL
		}
		fmt.Fprintf(w, `<url>
<loc>%s</loc>
<lastmod>%s</lastmod>
<changefreq>daily</changefreq>
<priority>0.8</priority>
</url>
`,
			link,
			job.UpdatedAt.Format("2006-01-02"),
		)
	}

	fmt.Fprint(w, `</urlset>`)
}
