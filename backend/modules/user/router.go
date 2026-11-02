package user

import (
	"github.com/go-chi/chi/v5"
)

// RegisterRoutes registers user module endpoints onto a chi.Router
func RegisterRoutes(r chi.Router, h *Handler) {
	r.Get("/", h.GetUsers)
	r.Post("/", h.CreateUser)
}

// Router returns an isolated chi.Router for the user module
func (h *Handler) Router() chi.Router {
	r := chi.NewRouter()
	RegisterRoutes(r, h)
	return r
}
