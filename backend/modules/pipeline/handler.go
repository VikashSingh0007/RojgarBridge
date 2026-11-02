package pipeline

import (
	"net/http"

	"rojgarbridge-backend/utils"
)

type Handler struct {
	service Service
}

func NewHandler(svc Service) *Handler {
	return &Handler{service: svc}
}

// TriggerPipeline - manual trigger to scrape and extract
func (h *Handler) TriggerPipeline(w http.ResponseWriter, r *http.Request) {
	go func() {
		h.service.RunPipeline()
	}()

	utils.RespondJSON(w, http.StatusAccepted, map[string]string{
		"message": "Scraping and AI extraction pipeline triggered in background across official government portals",
	})
}

// ReExtractDrafts - re-runs improved extraction on existing drafts
func (h *Handler) ReExtractDrafts(w http.ResponseWriter, r *http.Request) {
	go func() {
		h.service.ReExtractDrafts()
	}()

	utils.RespondJSON(w, http.StatusAccepted, map[string]string{
		"message": "Re-extraction started for existing draft jobs in background",
	})
}
