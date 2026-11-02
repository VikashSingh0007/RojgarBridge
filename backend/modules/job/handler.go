package job

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"rojgarbridge-backend/utils"
)

type Handler struct {
	service Service
	cache   *utils.Cache
}

func NewHandler(svc Service, cache *utils.Cache) *Handler {
	return &Handler{
		service: svc,
		cache:   cache,
	}
}

// GetAllJobs - lists all jobs (draft + published) with optional filters
func (h *Handler) GetAllJobs(w http.ResponseWriter, r *http.Request) {
	limit := utils.ParseLimit(r.URL.Query().Get("limit"), 100)
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	if offset < 0 {
		offset = 0
	}
	status := strings.TrimSpace(r.URL.Query().Get("status"))
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	sortBy := strings.TrimSpace(r.URL.Query().Get("sort"))
	categories := utils.GetQuerySlice(r, "category")
	qualifications := utils.GetQuerySlice(r, "qualification")
	states := utils.GetQuerySlice(r, "state")
	noticeTypes := utils.GetQuerySlice(r, "type")

	jobs, total, err := h.service.QueryJobs(JobFilterParams{
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
		utils.RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	utils.RespondJSON(w, http.StatusOK, map[string]interface{}{
		"total":  total,
		"count":  len(jobs),
		"limit":  limit,
		"offset": offset,
		"jobs":   jobs,
	})
}

// GetDrafts - lists all jobs awaiting review
func (h *Handler) GetDrafts(w http.ResponseWriter, r *http.Request) {
	limit := utils.ParseLimit(r.URL.Query().Get("limit"), 100)
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	if offset < 0 {
		offset = 0
	}
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	sortBy := strings.TrimSpace(r.URL.Query().Get("sort"))
	categories := utils.GetQuerySlice(r, "category")
	qualifications := utils.GetQuerySlice(r, "qualification")
	states := utils.GetQuerySlice(r, "state")
	noticeTypes := utils.GetQuerySlice(r, "type")

	jobs, total, err := h.service.QueryJobs(JobFilterParams{
		Status:         string(StatusDraft),
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
		utils.RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	utils.RespondJSON(w, http.StatusOK, map[string]interface{}{
		"total":  total,
		"count":  len(jobs),
		"limit":  limit,
		"offset": offset,
		"jobs":   jobs,
	})
}

// GetPublished - lists live jobs for public viewing with full DB filtering (cached)
func (h *Handler) GetPublished(w http.ResponseWriter, r *http.Request) {
	cacheKey := "published:" + r.URL.RawQuery
	if h.cache != nil {
		if cached, ok := h.cache.Get(cacheKey); ok {
			utils.RespondJSON(w, http.StatusOK, cached)
			return
		}
	}

	limit := utils.ParseLimit(r.URL.Query().Get("limit"), 10)
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	if offset < 0 {
		offset = 0
	}
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	sortBy := strings.TrimSpace(r.URL.Query().Get("sort"))
	categories := utils.GetQuerySlice(r, "category")
	qualifications := utils.GetQuerySlice(r, "qualification")
	states := utils.GetQuerySlice(r, "state")
	noticeTypes := utils.GetQuerySlice(r, "type")

	jobs, total, err := h.service.QueryJobs(JobFilterParams{
		Status:         string(StatusPublished),
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
		utils.RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	resp := map[string]interface{}{
		"total":  total,
		"count":  len(jobs),
		"limit":  limit,
		"offset": offset,
		"jobs":   jobs,
	}
	if h.cache != nil {
		h.cache.Set(cacheKey, resp, 30*time.Second)
	}
	utils.RespondJSON(w, http.StatusOK, resp)
}

// SearchJobs - full text search with qualification, state, category, type and sort filters (cached)
func (h *Handler) SearchJobs(w http.ResponseWriter, r *http.Request) {
	cacheKey := "search:" + r.URL.RawQuery
	if h.cache != nil {
		if cached, ok := h.cache.Get(cacheKey); ok {
			utils.RespondJSON(w, http.StatusOK, cached)
			return
		}
	}

	q := strings.TrimSpace(r.URL.Query().Get("q"))
	status := strings.TrimSpace(r.URL.Query().Get("status"))
	if status == "" {
		status = string(StatusPublished)
	}
	limit := utils.ParseLimit(r.URL.Query().Get("limit"), 10)
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	if offset < 0 {
		offset = 0
	}
	sortBy := strings.TrimSpace(r.URL.Query().Get("sort"))
	categories := utils.GetQuerySlice(r, "category")
	qualifications := utils.GetQuerySlice(r, "qualification")
	states := utils.GetQuerySlice(r, "state")
	noticeTypes := utils.GetQuerySlice(r, "type")

	jobs, total, err := h.service.QueryJobs(JobFilterParams{
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
		utils.RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	resp := map[string]interface{}{
		"query":  q,
		"total":  total,
		"count":  len(jobs),
		"limit":  limit,
		"offset": offset,
		"jobs":   jobs,
	}
	if h.cache != nil {
		h.cache.Set(cacheKey, resp, 30*time.Second)
	}
	utils.RespondJSON(w, http.StatusOK, resp)
}

// GetClosingSoon - returns jobs expiring in the upcoming days (cached)
func (h *Handler) GetClosingSoon(w http.ResponseWriter, r *http.Request) {
	days, _ := strconv.Atoi(r.URL.Query().Get("days"))
	if days <= 0 {
		days = 7
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit <= 0 {
		limit = 50
	}

	cacheKey := fmt.Sprintf("closing_soon:%d:%d", days, limit)
	if h.cache != nil {
		if cached, ok := h.cache.Get(cacheKey); ok {
			utils.RespondJSON(w, http.StatusOK, cached)
			return
		}
	}

	jobs, err := h.service.GetClosingSoonJobs(days, limit)
	if err != nil {
		utils.RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	resp := map[string]interface{}{
		"days_window": days,
		"count":       len(jobs),
		"jobs":        jobs,
	}
	if h.cache != nil {
		h.cache.Set(cacheKey, resp, 60*time.Second)
	}
	utils.RespondJSON(w, http.StatusOK, resp)
}

// GetAnalytics - returns aggregated dashboard metrics (cached)
func (h *Handler) GetAnalytics(w http.ResponseWriter, r *http.Request) {
	const cacheKey = "analytics"
	if h.cache != nil {
		if cached, ok := h.cache.Get(cacheKey); ok {
			utils.RespondJSON(w, http.StatusOK, cached)
			return
		}
	}

	analytics, err := h.service.GetAnalytics()
	if err != nil {
		utils.RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if h.cache != nil {
		h.cache.Set(cacheKey, analytics, 60*time.Second)
	}
	utils.RespondJSON(w, http.StatusOK, analytics)
}

// GetJobByID - details of a single job
func (h *Handler) GetJobByID(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		utils.RespondError(w, http.StatusBadRequest, "invalid job id")
		return
	}

	job, err := h.service.GetJobByID(id)
	if err != nil {
		utils.RespondError(w, http.StatusNotFound, "job not found")
		return
	}

	utils.RespondJSON(w, http.StatusOK, job)
}

// UpdateJob - edit job details prior to publishing
func (h *Handler) UpdateJob(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		utils.RespondError(w, http.StatusBadRequest, "invalid job id")
		return
	}

	job, err := h.service.GetJobByID(id)
	if err != nil {
		utils.RespondError(w, http.StatusNotFound, "job not found")
		return
	}

	var payload UpdateJobPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		utils.RespondError(w, http.StatusBadRequest, "malformed request payload: "+err.Error())
		return
	}

	job.Details = payload.Details
	if payload.ReviewNotes != "" {
		job.ReviewNotes = payload.ReviewNotes
	}

	if err := h.service.UpdateJob(job); err != nil {
		utils.RespondError(w, http.StatusInternalServerError, "failed to update job: "+err.Error())
		return
	}
	if h.cache != nil {
		h.cache.Clear()
	}

	utils.RespondJSON(w, http.StatusOK, map[string]interface{}{
		"message": "job updated successfully",
		"job":     job,
	})
}

// PublishJob - approve draft and publish
func (h *Handler) PublishJob(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		utils.RespondError(w, http.StatusBadRequest, "invalid job id")
		return
	}

	_, err = h.service.PublishJob(id)
	if err != nil {
		utils.RespondError(w, http.StatusInternalServerError, "failed to publish job: "+err.Error())
		return
	}
	if h.cache != nil {
		h.cache.Clear()
	}

	utils.RespondJSON(w, http.StatusOK, map[string]interface{}{
		"message": "job published successfully",
		"id":      id,
		"status":  StatusPublished,
	})
}

// BulkPublish - publish all or selected drafts
func (h *Handler) BulkPublish(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		IDs []int64 `json:"ids"`
	}
	_ = json.NewDecoder(r.Body).Decode(&payload)

	count, err := h.service.BulkPublish(payload.IDs)
	if err != nil {
		utils.RespondError(w, http.StatusInternalServerError, "failed to bulk publish: "+err.Error())
		return
	}
	if h.cache != nil {
		h.cache.Clear()
	}

	utils.RespondJSON(w, http.StatusOK, map[string]interface{}{
		"message":         "Jobs published successfully",
		"published_count": count,
	})
}

// RejectJob - reject draft
func (h *Handler) RejectJob(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		utils.RespondError(w, http.StatusBadRequest, "invalid job id")
		return
	}

	var payload struct {
		Reason string `json:"reason"`
	}
	_ = json.NewDecoder(r.Body).Decode(&payload)
	if payload.Reason == "" {
		payload.Reason = "Rejected by admin"
	}

	if err := h.service.RejectJob(id, payload.Reason); err != nil {
		utils.RespondError(w, http.StatusInternalServerError, "failed to reject job: "+err.Error())
		return
	}
	if h.cache != nil {
		h.cache.Clear()
	}

	utils.RespondJSON(w, http.StatusOK, map[string]interface{}{
		"message": "job rejected",
		"id":      id,
		"status":  StatusRejected,
	})
}

// GetRSSFeed - serves standard RSS 2.0 XML for Google SEO
func (h *Handler) GetRSSFeed(w http.ResponseWriter, r *http.Request) {
	jobs, err := h.service.GetRecentPublished(50)
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
	jobs, err := h.service.GetRecentPublished(200)
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
