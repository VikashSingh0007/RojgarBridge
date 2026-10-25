package api

import (
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
)

func NewRouter(h *Handler) *chi.Mux {
	r := chi.NewRouter()

	// Standard middleware
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	// Cross-Origin Resource Sharing (CORS) for future frontend
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: false,
		MaxAge:           300,
	}))

	r.Get("/", h.Health)
	r.Head("/", h.Health)
	r.Get("/health", h.Health)

	r.Route("/api", func(r chi.Router) {
		r.Route("/jobs", func(r chi.Router) {
			r.Get("/drafts", h.GetDrafts)
			r.Get("/published", h.GetPublished)
			r.Get("/{id}", h.GetJobByID)
			r.Put("/{id}", h.UpdateJob)
			r.Post("/{id}/publish", h.PublishJob)
			r.Post("/{id}/reject", h.RejectJob)
		})

		r.Post("/pipeline/run", h.TriggerPipeline)
		r.Post("/pipeline/reextract", h.ReExtractDrafts)
	})

	return r
}
