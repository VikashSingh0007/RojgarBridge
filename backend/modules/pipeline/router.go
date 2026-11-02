package pipeline

import (
	"github.com/go-chi/chi/v5"
)

// RegisterRoutes registers pipeline module endpoints onto a chi.Router
func RegisterRoutes(r chi.Router, h *Handler) {
	r.Post("/run", h.TriggerPipeline)
	r.Post("/reextract", h.ReExtractDrafts)
}

// Router returns an isolated chi.Router for the pipeline module
func (h *Handler) Router() chi.Router {
	r := chi.NewRouter()
	RegisterRoutes(r, h)
	return r
}
