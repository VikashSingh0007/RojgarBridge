package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"rojgarbridge-backend/internal/db"
	"rojgarbridge-backend/internal/models"
	"rojgarbridge-backend/internal/scrapers"
)

type Handler struct {
	db      *db.DB
	manager *scrapers.Manager
}

func NewHandler(database *db.DB, manager *scrapers.Manager) *Handler {
	return &Handler{
		db:      database,
		manager: manager,
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
		"service": "RojgarBridge Core Backend",
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

// GetAllJobs - lists all jobs (draft + published) with optional filters
func (h *Handler) GetAllJobs(w http.ResponseWriter, r *http.Request) {
	limit := parseLimit(r.URL.Query().Get("limit"), 100)
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	noticeType := r.URL.Query().Get("type")
	category := r.URL.Query().Get("category")
	state := r.URL.Query().Get("state")
	status := r.URL.Query().Get("status")

	jobs, total, err := h.db.GetJobs(status, noticeType, category, state, limit, offset)
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
	noticeType := r.URL.Query().Get("type")
	category := r.URL.Query().Get("category")
	state := r.URL.Query().Get("state")

	jobs, total, err := h.db.GetJobs(string(models.StatusDraft), noticeType, category, state, limit, offset)
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

// GetPublished - lists live jobs for the public page
func (h *Handler) GetPublished(w http.ResponseWriter, r *http.Request) {
	limit := parseLimit(r.URL.Query().Get("limit"), 100)
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	noticeType := r.URL.Query().Get("type")
	category := r.URL.Query().Get("category")
	state := r.URL.Query().Get("state")

	jobs, total, err := h.db.GetJobs(string(models.StatusPublished), noticeType, category, state, limit, offset)
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
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	job.Details = payload.Details
	job.ReviewNotes = payload.ReviewNotes

	if err := h.db.UpdateJob(job); err != nil {
		respondError(w, http.StatusInternalServerError, "failed to update job: "+err.Error())
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"message": "job updated successfully",
		"job":     job,
	})
}

// PublishJob - approve draft and make live
func (h *Handler) PublishJob(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid job id")
		return
	}

	if err := h.db.PublishJob(id); err != nil {
		respondError(w, http.StatusInternalServerError, "failed to publish job: "+err.Error())
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"message": "job published successfully",
		"id":      id,
		"status":  models.StatusPublished,
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
	// Run in background
	go func() {
		h.manager.RunPipeline()
	}()

	respondJSON(w, http.StatusAccepted, map[string]string{
		"message": "Scraping and AI extraction pipeline triggered in background across all 12 portal sources",
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
