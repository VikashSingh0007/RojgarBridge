package auth

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"rojgarbridge-backend/config"
	"rojgarbridge-backend/middleware"
	"rojgarbridge-backend/utils"
)

type Handler struct {
	service Service
	cfg     *config.Config
}

func NewHandler(svc Service, cfg *config.Config) *Handler {
	return &Handler{
		service: svc,
		cfg:     cfg,
	}
}

// Login handles authentication for administrators and registered users
func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.RespondError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	resp, err := h.service.Login(req.Email, req.Password)
	if err != nil {
		if errorsIs(err, ErrInvalidCredentials) {
			utils.RespondError(w, http.StatusUnauthorized, "Invalid email or password")
			return
		}
		utils.RespondError(w, http.StatusBadRequest, err.Error())
		return
	}

	utils.RespondJSON(w, http.StatusOK, resp)
}

func errorsIs(err, target error) bool {
	return err != nil && target != nil && err.Error() == target.Error()
}

// GetAuthMe returns the current authenticated user's profile
func (h *Handler) GetAuthMe(w http.ResponseWriter, r *http.Request) {
	authHeader := r.Header.Get("Authorization")
	u, err := h.service.GetAuthMe(authHeader)
	if err != nil {
		utils.RespondError(w, http.StatusUnauthorized, "Not authenticated")
		return
	}

	utils.RespondJSON(w, http.StatusOK, u)
}

// GoogleLogin redirects user to Google's OAuth 2.0 consent page
func (h *Handler) GoogleLogin(w http.ResponseWriter, r *http.Request) {
	frontendURL := middleware.DetectFrontendURL(r, h.cfg)

	if h.cfg == nil || strings.TrimSpace(h.cfg.GoogleClientID) == "" {
		http.Redirect(w, r, frontendURL+"/auth/callback?error=google_client_id_not_configured", http.StatusTemporaryRedirect)
		return
	}

	callbackURL := middleware.DetectBackendCallbackURL(r, h.cfg)

	googleAuthURL := fmt.Sprintf(
		"https://accounts.google.com/o/oauth2/v2/auth?client_id=%s&redirect_uri=%s&response_type=code&scope=%s&access_type=offline&prompt=select_account&state=%s",
		url.QueryEscape(h.cfg.GoogleClientID),
		url.QueryEscape(callbackURL),
		url.QueryEscape("openid email profile"),
		url.QueryEscape(frontendURL),
	)

	http.Redirect(w, r, googleAuthURL, http.StatusTemporaryRedirect)
}

// GoogleCallback handles the redirect callback from Google with authorization code
func (h *Handler) GoogleCallback(w http.ResponseWriter, r *http.Request) {
	frontendURL := middleware.DetectFrontendURL(r, h.cfg)
	if state := r.URL.Query().Get("state"); state != "" {
		if unescaped, err := url.QueryUnescape(state); err == nil && (strings.HasPrefix(unescaped, "http://") || strings.HasPrefix(unescaped, "https://")) {
			frontendURL = strings.TrimRight(unescaped, "/")
		} else if strings.HasPrefix(state, "http://") || strings.HasPrefix(state, "https://") {
			frontendURL = strings.TrimRight(state, "/")
		}
	}

	errParam := r.URL.Query().Get("error")
	if errParam != "" {
		http.Redirect(w, r, frontendURL+"/auth/callback?error="+url.QueryEscape(errParam), http.StatusTemporaryRedirect)
		return
	}

	code := r.URL.Query().Get("code")
	if code == "" {
		http.Redirect(w, r, frontendURL+"/auth/callback?error=missing_auth_code", http.StatusTemporaryRedirect)
		return
	}

	callbackURL := middleware.DetectBackendCallbackURL(r, h.cfg)

	gUser, err := h.service.ExchangeGoogleCode(code, callbackURL)
	if err != nil {
		http.Redirect(w, r, frontendURL+"/auth/callback?error=google_auth_failed", http.StatusTemporaryRedirect)
		return
	}

	user, sessionToken, err := h.service.FindOrCreateGoogleUser(gUser)
	if err != nil {
		http.Redirect(w, r, frontendURL+"/auth/callback?error=db_save_failed", http.StatusTemporaryRedirect)
		return
	}

	userJSON, _ := json.Marshal(user)
	redirectURL := fmt.Sprintf(
		"%s/auth/callback?token=%s&user=%s",
		frontendURL,
		url.QueryEscape(sessionToken),
		url.QueryEscape(string(userJSON)),
	)

	http.Redirect(w, r, redirectURL, http.StatusTemporaryRedirect)
}

// GoogleVerifyToken verifies client token or handles simulated Google login
func (h *Handler) GoogleVerifyToken(w http.ResponseWriter, r *http.Request) {
	var req GoogleVerifyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.RespondError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	user, token, err := h.service.VerifyGoogleToken(req.AccessToken, req.MockUser)
	if err != nil {
		if err.Error() == "invalid Google access token" {
			utils.RespondError(w, http.StatusUnauthorized, "Invalid Google access token")
			return
		}
		utils.RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	utils.RespondJSON(w, http.StatusOK, LoginResponse{
		Success: true,
		Token:   token,
		User:    *user,
		Message: "Google authentication successful",
	})
}
