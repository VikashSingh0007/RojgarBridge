package job

import (
	"github.com/go-chi/chi/v5"
)

// RegisterRoutes registers job module endpoints onto a chi.Router
func RegisterRoutes(r chi.Router, h *Handler) {
	r.Get("/", h.GetAllJobs)
	r.Get("/search", h.SearchJobs)
	r.Get("/closing-soon", h.GetClosingSoon)
	r.Get("/analytics", h.GetAnalytics)
	r.Get("/drafts", h.GetDrafts)
	r.Get("/published", h.GetPublished)
	r.Get("/{id}", h.GetJobByID)
	r.Put("/{id}", h.UpdateJob)
	r.Post("/{id}/publish", h.PublishJob)
	r.Post("/bulk-publish", h.BulkPublish)
	r.Post("/{id}/reject", h.RejectJob)
}

// RegisterFeedRoutes registers public SEO XML feeds
func RegisterFeedRoutes(r chi.Router, h *Handler) {
	r.Get("/feed.xml", h.GetRSSFeed)
	r.Get("/rss.xml", h.GetRSSFeed)
	r.Get("/sitemap.xml", h.GetSitemap)
}

// Router returns an isolated chi.Router for the job module
func (h *Handler) Router() chi.Router {
	r := chi.NewRouter()
	RegisterRoutes(r, h)
	return r
}
