package auth

import (
	"github.com/go-chi/chi/v5"
)

// RegisterRoutes registers auth module endpoints onto a chi.Router
func RegisterRoutes(r chi.Router, h *Handler) {
	r.Post("/login", h.Login)
	r.Get("/me", h.GetAuthMe)
	r.Get("/google/login", h.GoogleLogin)
	r.Get("/google/callback", h.GoogleCallback)
	r.Post("/google/verify", h.GoogleVerifyToken)
}

// Router returns an isolated chi.Router for the auth module
func (h *Handler) Router() chi.Router {
	r := chi.NewRouter()
	RegisterRoutes(r, h)
	return r
}
