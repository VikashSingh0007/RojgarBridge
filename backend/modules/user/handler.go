package user

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"rojgarbridge-backend/utils"
)

type Handler struct {
	service Service
}

func NewHandler(svc Service) *Handler {
	return &Handler{service: svc}
}

// GetUsers retrieves all users from database
func (h *Handler) GetUsers(w http.ResponseWriter, r *http.Request) {
	limit := utils.ParseLimit(r.URL.Query().Get("limit"), 50)
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	if offset < 0 {
		offset = 0
	}

	users, total, err := h.service.ListUsers(limit, offset)
	if err != nil {
		utils.RespondError(w, http.StatusInternalServerError, "Failed to retrieve users: "+err.Error())
		return
	}

	utils.RespondJSON(w, http.StatusOK, map[string]interface{}{
		"users":  users,
		"total":  total,
		"limit":  limit,
		"offset": offset,
	})
}

// CreateUser handles registration / adding a new user into the database
func (h *Handler) CreateUser(w http.ResponseWriter, r *http.Request) {
	var req CreateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.RespondError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	email := strings.TrimSpace(req.Email)
	password := strings.TrimSpace(req.Password)
	name := strings.TrimSpace(req.Name)

	if email == "" || password == "" {
		utils.RespondError(w, http.StatusBadRequest, "Email and password are required")
		return
	}

	if name == "" {
		name = strings.Split(email, "@")[0]
	}

	existing, _ := h.service.GetUserByEmail(email)
	if existing != nil {
		utils.RespondError(w, http.StatusConflict, "User with this email already exists")
		return
	}

	role := RoleUser
	if strings.ToLower(req.Role) == "admin" {
		role = RoleAdmin
	}

	newUser := &User{
		Name:  name,
		Email: email,
		Role:  role,
		Phone: strings.TrimSpace(req.Phone),
	}

	if err := h.service.CreateUser(newUser, password); err != nil {
		utils.RespondError(w, http.StatusInternalServerError, "Failed to create user: "+err.Error())
		return
	}

	utils.RespondJSON(w, http.StatusCreated, map[string]interface{}{
		"message": "User registered successfully",
		"user":    newUser,
	})
}
