package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"golang.org/x/crypto/bcrypt"
	"rojgarbridge-backend/internal/config"
	"rojgarbridge-backend/internal/db"
	"rojgarbridge-backend/internal/models"
	"rojgarbridge-backend/internal/notifier"
	"rojgarbridge-backend/internal/scrapers"
)

type Handler struct {
	db       *db.DB
	manager  *scrapers.Manager
	notifier notifier.Notifier
	cfg      *config.Config
}

func NewHandler(database *db.DB, manager *scrapers.Manager, notif notifier.Notifier, cfg *config.Config) *Handler {
	return &Handler{
		db:       database,
		manager:  manager,
		notifier: notif,
		cfg:      cfg,
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
		"service": "RojgarBridge Official Government Job Engine",
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

func getQuerySlice(r *http.Request, key string) []string {
	values := r.URL.Query()[key]
	var result []string
	for _, v := range values {
		parts := strings.Split(v, ",")
		for _, p := range parts {
			trimmed := strings.TrimSpace(p)
			if trimmed != "" && !strings.EqualFold(trimmed, "all") {
				result = append(result, trimmed)
			}
		}
	}
	return result
}

// GetAllJobs - lists all jobs (draft + published) with optional filters
func (h *Handler) GetAllJobs(w http.ResponseWriter, r *http.Request) {
	limit := parseLimit(r.URL.Query().Get("limit"), 100)
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	if offset < 0 {
		offset = 0
	}
	status := strings.TrimSpace(r.URL.Query().Get("status"))
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	sortBy := strings.TrimSpace(r.URL.Query().Get("sort"))
	categories := getQuerySlice(r, "category")
	qualifications := getQuerySlice(r, "qualification")
	states := getQuerySlice(r, "state")
	noticeTypes := getQuerySlice(r, "type")

	jobs, total, err := h.db.QueryJobs(db.JobFilterParams{
		Status:         status,
		Query:          q,
		NoticeTypes:    noticeTypes,
		Categories:     categories,
		Qualifications: qualifications,
		States:         states,
		SortBy:         sortBy,
		Limit:          limit,
		Offset:         offset,
	})
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
	if offset < 0 {
		offset = 0
	}
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	sortBy := strings.TrimSpace(r.URL.Query().Get("sort"))
	categories := getQuerySlice(r, "category")
	qualifications := getQuerySlice(r, "qualification")
	states := getQuerySlice(r, "state")
	noticeTypes := getQuerySlice(r, "type")

	jobs, total, err := h.db.QueryJobs(db.JobFilterParams{
		Status:         string(models.StatusDraft),
		Query:          q,
		NoticeTypes:    noticeTypes,
		Categories:     categories,
		Qualifications: qualifications,
		States:         states,
		SortBy:         sortBy,
		Limit:          limit,
		Offset:         offset,
	})
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

// GetPublished - lists live jobs for public viewing with full DB filtering
func (h *Handler) GetPublished(w http.ResponseWriter, r *http.Request) {
	limit := parseLimit(r.URL.Query().Get("limit"), 10)
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	if offset < 0 {
		offset = 0
	}
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	sortBy := strings.TrimSpace(r.URL.Query().Get("sort"))
	categories := getQuerySlice(r, "category")
	qualifications := getQuerySlice(r, "qualification")
	states := getQuerySlice(r, "state")
	noticeTypes := getQuerySlice(r, "type")

	jobs, total, err := h.db.QueryJobs(db.JobFilterParams{
		Status:         string(models.StatusPublished),
		Query:          q,
		NoticeTypes:    noticeTypes,
		Categories:     categories,
		Qualifications: qualifications,
		States:         states,
		SortBy:         sortBy,
		Limit:          limit,
		Offset:         offset,
	})
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

// SearchJobs - full text search with qualification, state, category, type and sort filters
func (h *Handler) SearchJobs(w http.ResponseWriter, r *http.Request) {
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	status := strings.TrimSpace(r.URL.Query().Get("status"))
	if status == "" {
		status = string(models.StatusPublished)
	}
	limit := parseLimit(r.URL.Query().Get("limit"), 10)
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	if offset < 0 {
		offset = 0
	}
	sortBy := strings.TrimSpace(r.URL.Query().Get("sort"))
	categories := getQuerySlice(r, "category")
	qualifications := getQuerySlice(r, "qualification")
	states := getQuerySlice(r, "state")
	noticeTypes := getQuerySlice(r, "type")

	jobs, total, err := h.db.QueryJobs(db.JobFilterParams{
		Status:         status,
		Query:          q,
		NoticeTypes:    noticeTypes,
		Categories:     categories,
		Qualifications: qualifications,
		States:         states,
		SortBy:         sortBy,
		Limit:          limit,
		Offset:         offset,
	})
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"query":  q,
		"total":  total,
		"count":  len(jobs),
		"limit":  limit,
		"offset": offset,
		"jobs":   jobs,
	})
}

// GetClosingSoon - returns jobs expiring in the upcoming days
func (h *Handler) GetClosingSoon(w http.ResponseWriter, r *http.Request) {
	days, _ := strconv.Atoi(r.URL.Query().Get("days"))
	if days <= 0 {
		days = 7
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit <= 0 {
		limit = 50
	}

	jobs, err := h.db.GetClosingSoonJobs(days, limit)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"days_window": days,
		"count":       len(jobs),
		"jobs":        jobs,
	})
}

// GetAnalytics - returns aggregated dashboard metrics
func (h *Handler) GetAnalytics(w http.ResponseWriter, r *http.Request) {
	analytics, err := h.db.GetAnalytics()
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respondJSON(w, http.StatusOK, analytics)
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
		respondError(w, http.StatusBadRequest, "malformed request payload: "+err.Error())
		return
	}

	job.Details = payload.Details
	if payload.ReviewNotes != "" {
		job.ReviewNotes = payload.ReviewNotes
	}

	if err := h.db.UpdateJob(job); err != nil {
		respondError(w, http.StatusInternalServerError, "failed to update job: "+err.Error())
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"message": "job updated successfully",
		"job":     job,
	})
}

// PublishJob - approve draft and publish
func (h *Handler) PublishJob(w http.ResponseWriter, r *http.Request) {
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

	if err := h.db.PublishJob(id); err != nil {
		respondError(w, http.StatusInternalServerError, "failed to publish job: "+err.Error())
		return
	}

	job.Status = models.StatusPublished

	// Broadcast notification to Telegram
	if h.notifier != nil {
		go func(j *models.Job) {
			_ = h.notifier.NotifyJobPublished(j)
		}(job)
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"message": "job published successfully",
		"id":      id,
		"status":  models.StatusPublished,
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
	go func() {
		h.manager.RunPipeline()
	}()

	respondJSON(w, http.StatusAccepted, map[string]string{
		"message": "Scraping and AI extraction pipeline triggered in background across official government portals",
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

// GetRSSFeed - serves standard RSS 2.0 XML for Google SEO
func (h *Handler) GetRSSFeed(w http.ResponseWriter, r *http.Request) {
	jobs, err := h.db.GetRecentPublished(50)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/rss+xml; charset=utf-8")
	fmt.Fprintf(w, `<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0" xmlns:atom="http://www.w3.org/2005/Atom">
<channel>
<title>RojgarHub - Official Sarkari Naukri &amp; Govt Job Alerts</title>
<link>https://rojgarhub.in/</link>
<description>Latest verified Indian government jobs, exam notices, and admit cards directly from official portals.</description>
<language>en-in</language>
<lastBuildDate>%s</lastBuildDate>
`, time.Now().Format(time.RFC1123Z))

	for _, job := range jobs {
		link := job.Details.OfficialApplyURL
		if link == "" {
			link = job.RawNotice.NoticeURL
		}
		fmt.Fprintf(w, `<item>
<title><![CDATA[%s]]></title>
<link>%s</link>
<guid isPermaLink="false">%s</guid>
<description><![CDATA[Organization: %s | Category: %s | Vacancies: %d | Eligibility: %s | Last Date: %s]]></description>
<pubDate>%s</pubDate>
</item>
`,
			job.Details.Title,
			link,
			job.Hash,
			job.Details.Organization,
			job.Details.Category,
			job.Details.TotalVacancies,
			job.Details.Qualification,
			job.Details.ApplicationEnd,
			job.CreatedAt.Format(time.RFC1123Z),
		)
	}

	fmt.Fprint(w, `</channel>
</rss>`)
}

// GetSitemap - serves XML sitemap for SEO crawlers
func (h *Handler) GetSitemap(w http.ResponseWriter, r *http.Request) {
	jobs, err := h.db.GetRecentPublished(200)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/xml; charset=utf-8")
	fmt.Fprint(w, `<?xml version="1.0" encoding="UTF-8"?>
<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
<url>
<loc>https://rojgarhub.in/</loc>
<changefreq>hourly</changefreq>
<priority>1.0</priority>
</url>
`)

	for _, job := range jobs {
		link := job.Details.OfficialApplyURL
		if link == "" {
			link = job.RawNotice.NoticeURL
		}
		fmt.Fprintf(w, `<url>
<loc>%s</loc>
<lastmod>%s</lastmod>
<changefreq>daily</changefreq>
<priority>0.8</priority>
</url>
`,
			link,
			job.UpdatedAt.Format("2006-01-02"),
		)
	}

	fmt.Fprint(w, `</urlset>`)
}

// Login handles authentication for administrators and registered users
func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var req models.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	email := strings.TrimSpace(req.Email)
	password := strings.TrimSpace(req.Password)

	if email == "" || password == "" {
		respondError(w, http.StatusBadRequest, "Email and password are required")
		return
	}

	// 1. Explicit hardcoded admin credential support
	if strings.EqualFold(email, "admin@admin.com") && password == "admin@7077" {
		adminUser, err := h.db.GetUserByEmail("admin@admin.com")
		if err != nil || adminUser == nil {
			adminUser = &models.User{
				ID:        1,
				Name:      "System Administrator",
				Email:     "admin@admin.com",
				Role:      models.RoleAdmin,
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			}
		}
		respondJSON(w, http.StatusOK, models.LoginResponse{
			Success: true,
			Token:   "admin-session-" + strconv.FormatInt(time.Now().Unix(), 10),
			User:    *adminUser,
			Message: "Admin authenticated successfully",
		})
		return
	}

	// 2. Lookup in database
	user, err := h.db.GetUserByEmail(email)
	if err != nil || user == nil {
		respondError(w, http.StatusUnauthorized, "Invalid email or password")
		return
	}

	// Compare password hash or fallback
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		if user.PasswordHash != password {
			respondError(w, http.StatusUnauthorized, "Invalid email or password")
			return
		}
	}

	respondJSON(w, http.StatusOK, models.LoginResponse{
		Success: true,
		Token:   fmt.Sprintf("user-session-%d-%d", user.ID, time.Now().Unix()),
		User:    *user,
		Message: "Authentication successful",
	})
}

// GetAuthMe returns the current authenticated user's profile
func (h *Handler) GetAuthMe(w http.ResponseWriter, r *http.Request) {
	authHeader := r.Header.Get("Authorization")
	if strings.Contains(authHeader, "admin") {
		adminUser, _ := h.db.GetUserByEmail("admin@admin.com")
		if adminUser != nil {
			respondJSON(w, http.StatusOK, adminUser)
			return
		}
		respondJSON(w, http.StatusOK, models.User{
			ID:        1,
			Name:      "System Administrator",
			Email:     "admin@admin.com",
			Role:      models.RoleAdmin,
			CreatedAt: time.Now(),
		})
		return
	}

	adminUser, err := h.db.GetUserByEmail("admin@admin.com")
	if err == nil && adminUser != nil {
		respondJSON(w, http.StatusOK, adminUser)
		return
	}

	respondError(w, http.StatusUnauthorized, "Not authenticated")
}

// GetUsers retrieves all users from database
func (h *Handler) GetUsers(w http.ResponseWriter, r *http.Request) {
	limit := parseLimit(r.URL.Query().Get("limit"), 50)
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	if offset < 0 {
		offset = 0
	}

	users, total, err := h.db.ListUsers(limit, offset)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to retrieve users: "+err.Error())
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"users":  users,
		"total":  total,
		"limit":  limit,
		"offset": offset,
	})
}

// CreateUser handles registration / adding a new user into the database
func (h *Handler) CreateUser(w http.ResponseWriter, r *http.Request) {
	var req models.CreateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	email := strings.TrimSpace(req.Email)
	password := strings.TrimSpace(req.Password)
	name := strings.TrimSpace(req.Name)

	if email == "" || password == "" {
		respondError(w, http.StatusBadRequest, "Email and password are required")
		return
	}

	if name == "" {
		name = strings.Split(email, "@")[0]
	}

	existing, _ := h.db.GetUserByEmail(email)
	if existing != nil {
		respondError(w, http.StatusConflict, "User with this email already exists")
		return
	}

	role := models.RoleUser
	if strings.ToLower(req.Role) == "admin" {
		role = models.RoleAdmin
	}

	newUser := &models.User{
		Name:  name,
		Email: email,
		Role:  role,
		Phone: strings.TrimSpace(req.Phone),
	}

	if err := h.db.CreateUser(newUser, password); err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to create user: "+err.Error())
		return
	}

	respondJSON(w, http.StatusCreated, map[string]interface{}{
		"message": "User registered successfully",
		"user":    newUser,
	})
}

// GoogleLogin redirects user to Google's OAuth 2.0 consent page
func (h *Handler) GoogleLogin(w http.ResponseWriter, r *http.Request) {
	frontendURL := "http://localhost:3000"
	if h.cfg != nil && h.cfg.FrontendURL != "" {
		frontendURL = h.cfg.FrontendURL
	}

	if h.cfg == nil || strings.TrimSpace(h.cfg.GoogleClientID) == "" {
		// If Google credentials are not set in .env, redirect with helpful prompt
		http.Redirect(w, r, frontendURL+"/auth/callback?error=google_client_id_not_configured", http.StatusTemporaryRedirect)
		return
	}

	googleAuthURL := fmt.Sprintf(
		"https://accounts.google.com/o/oauth2/v2/auth?client_id=%s&redirect_uri=%s&response_type=code&scope=%s&access_type=offline&prompt=select_account&state=%s",
		url.QueryEscape(h.cfg.GoogleClientID),
		url.QueryEscape(h.cfg.GoogleRedirectURL),
		url.QueryEscape("openid email profile"),
		"state_rozgarhub",
	)

	http.Redirect(w, r, googleAuthURL, http.StatusTemporaryRedirect)
}

// GoogleCallback handles the redirect callback from Google with authorization code
func (h *Handler) GoogleCallback(w http.ResponseWriter, r *http.Request) {
	frontendURL := "http://localhost:3000"
	if h.cfg != nil && h.cfg.FrontendURL != "" {
		frontendURL = h.cfg.FrontendURL
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

	// 1. Exchange code for access token
	data := url.Values{}
	data.Set("code", code)
	data.Set("client_id", h.cfg.GoogleClientID)
	data.Set("client_secret", h.cfg.GoogleClientSecret)
	data.Set("redirect_uri", h.cfg.GoogleRedirectURL)
	data.Set("grant_type", "authorization_code")

	tokenResp, err := http.PostForm("https://oauth2.googleapis.com/token", data)
	if err != nil {
		http.Redirect(w, r, frontendURL+"/auth/callback?error=token_exchange_failed", http.StatusTemporaryRedirect)
		return
	}
	defer tokenResp.Body.Close()

	if tokenResp.StatusCode != http.StatusOK {
		http.Redirect(w, r, fmt.Sprintf("%s/auth/callback?error=google_auth_failed_%d", frontendURL, tokenResp.StatusCode), http.StatusTemporaryRedirect)
		return
	}

	var tokenData struct {
		AccessToken string `json:"access_token"`
		IDToken     string `json:"id_token"`
	}
	if err := json.NewDecoder(tokenResp.Body).Decode(&tokenData); err != nil {
		http.Redirect(w, r, frontendURL+"/auth/callback?error=invalid_token_response", http.StatusTemporaryRedirect)
		return
	}

	// 2. Fetch user profile from Google
	client := &http.Client{Timeout: 10 * time.Second}
	userReq, err := http.NewRequest("GET", "https://www.googleapis.com/oauth2/v2/userinfo", nil)
	if err != nil {
		http.Redirect(w, r, frontendURL+"/auth/callback?error=profile_request_failed", http.StatusTemporaryRedirect)
		return
	}
	userReq.Header.Set("Authorization", "Bearer "+tokenData.AccessToken)

	userResp, err := client.Do(userReq)
	if err != nil || userResp.StatusCode != http.StatusOK {
		http.Redirect(w, r, frontendURL+"/auth/callback?error=userinfo_fetch_failed", http.StatusTemporaryRedirect)
		return
	}
	defer userResp.Body.Close()

	var gUser models.GoogleUserInfo
	if err := json.NewDecoder(userResp.Body).Decode(&gUser); err != nil {
		http.Redirect(w, r, frontendURL+"/auth/callback?error=userinfo_decode_failed", http.StatusTemporaryRedirect)
		return
	}

	// 3. Find or Create User in database
	user, err := h.db.FindOrCreateGoogleUser(&gUser)
	if err != nil {
		http.Redirect(w, r, frontendURL+"/auth/callback?error=db_save_failed", http.StatusTemporaryRedirect)
		return
	}

	// 4. Generate Session Token & Redirect to Frontend
	sessionToken := fmt.Sprintf("google-session-%d-%d", user.ID, time.Now().Unix())
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
	var req struct {
		AccessToken string                 `json:"access_token"`
		IDToken     string                 `json:"id_token"`
		MockUser    *models.GoogleUserInfo `json:"mock_user,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	var gUser models.GoogleUserInfo

	if req.MockUser != nil && req.MockUser.Email != "" {
		gUser = *req.MockUser
	} else if req.AccessToken != "" {
		client := &http.Client{Timeout: 10 * time.Second}
		userReq, err := http.NewRequest("GET", "https://www.googleapis.com/oauth2/v2/userinfo", nil)
		if err != nil {
			respondError(w, http.StatusInternalServerError, "Failed to create request")
			return
		}
		userReq.Header.Set("Authorization", "Bearer "+req.AccessToken)
		userResp, err := client.Do(userReq)
		if err != nil || userResp.StatusCode != http.StatusOK {
			respondError(w, http.StatusUnauthorized, "Invalid Google access token")
			return
		}
		defer userResp.Body.Close()
		if err := json.NewDecoder(userResp.Body).Decode(&gUser); err != nil {
			respondError(w, http.StatusInternalServerError, "Failed to decode Google user")
			return
		}
	} else {
		respondError(w, http.StatusBadRequest, "access_token or mock_user required")
		return
	}

	user, err := h.db.FindOrCreateGoogleUser(&gUser)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to store user: "+err.Error())
		return
	}

	respondJSON(w, http.StatusOK, models.LoginResponse{
		Success: true,
		Token:   fmt.Sprintf("google-session-%d-%d", user.ID, time.Now().Unix()),
		User:    *user,
		Message: "Google authentication successful",
	})
}


