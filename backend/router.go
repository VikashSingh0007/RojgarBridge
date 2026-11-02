package main

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	chiMiddleware "github.com/go-chi/chi/v5/middleware"

	"rojgarbridge-backend/middleware"
	"rojgarbridge-backend/modules/auth"
	"rojgarbridge-backend/modules/job"
	"rojgarbridge-backend/modules/pipeline"
	"rojgarbridge-backend/modules/user"
	"rojgarbridge-backend/utils"
)

type ModuleRouters struct {
	AuthHandler     *auth.Handler
	UserHandler     *user.Handler
	JobHandler      *job.Handler
	PipelineHandler *pipeline.Handler
}

// NewRouter initializes the central router, registers middleware, and mounts all module routers
func NewRouter(handlers ModuleRouters) *chi.Mux {
	r := chi.NewRouter()

	// 1. Shared middleware
	r.Use(chiMiddleware.RequestID)
	r.Use(chiMiddleware.RealIP)
	r.Use(chiMiddleware.Logger)
	r.Use(chiMiddleware.Recoverer)
	r.Use(middleware.CORS())

	// 2. Health check endpoints
	r.Get("/", HealthCheckHandler)
	r.Head("/", HealthCheckHandler)
	r.Get("/health", HealthCheckHandler)

	// 3. Public SEO Feeds (registered through job module)
	job.RegisterFeedRoutes(r, handlers.JobHandler)

	// 4. Central API routing with dedicated module prefixes
	r.Route("/api", func(api chi.Router) {
		api.Route("/auth", func(sub chi.Router) {
			auth.RegisterRoutes(sub, handlers.AuthHandler)
		})
		api.Route("/users", func(sub chi.Router) {
			user.RegisterRoutes(sub, handlers.UserHandler)
		})
		api.Route("/jobs", func(sub chi.Router) {
			job.RegisterRoutes(sub, handlers.JobHandler)
		})
		api.Route("/pipeline", func(sub chi.Router) {
			pipeline.RegisterRoutes(sub, handlers.PipelineHandler)
		})
	})

	return r
}

func HealthCheckHandler(w http.ResponseWriter, r *http.Request) {
	utils.RespondJSON(w, http.StatusOK, map[string]string{
		"status":  "healthy",
		"service": "RojgarBridge Official Government Job Engine",
	})
}
