package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"rojgarbridge-backend/config"
	"rojgarbridge-backend/database"
	"rojgarbridge-backend/modules/auth"
	"rojgarbridge-backend/modules/job"
	"rojgarbridge-backend/modules/pipeline"
	"rojgarbridge-backend/modules/pipeline/extractor"
	"rojgarbridge-backend/modules/user"
	"rojgarbridge-backend/utils"
)

func setupTestRouter(t *testing.T) (http.Handler, *database.DB) {
	tempDB := filepath.Join(t.TempDir(), "api_test.db")
	dbConn, err := database.InitDB(tempDB, "")
	if err != nil {
		t.Fatalf("InitDB failed: %v", err)
	}

	cfg := &config.Config{
		FrontendURL:       "http://localhost:3000",
		GoogleRedirectURL: "http://localhost:8080/api/auth/google/callback",
	}

	appCache := utils.NewCache(2 * time.Minute)
	userService := user.NewService(dbConn)
	_ = userService.EnsureDefaultAdmin()

	jobService := job.NewService(dbConn, nil)
	_ = jobService.EnsurePublishedJobs()

	authService := auth.NewService(userService, cfg)
	ext := extractor.NewRuleBasedExtractor()
	pipeService := pipeline.NewService(jobService, ext)

	userHandler := user.NewHandler(userService)
	authHandler := auth.NewHandler(authService, cfg)
	jobHandler := job.NewHandler(jobService, appCache)
	pipeHandler := pipeline.NewHandler(pipeService)

	r := NewRouter(ModuleRouters{
		AuthHandler:     authHandler,
		UserHandler:     userHandler,
		JobHandler:      jobHandler,
		PipelineHandler: pipeHandler,
	})

	return r, dbConn
}

func TestAuthAndUserEndpoints(t *testing.T) {
	router, dbConn := setupTestRouter(t)
	defer dbConn.Close()

	// 1. Test Admin Login with hardcoded credentials
	body, _ := json.Marshal(auth.LoginRequest{
		Email:    "admin@admin.com",
		Password: "admin@7077",
	})
	req := httptest.NewRequest("POST", "/api/auth/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200 OK, got %d: %s", w.Code, w.Body.String())
	}

	var res auth.LoginResponse
	if err := json.Unmarshal(w.Body.Bytes(), &res); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}
	if !res.Success || res.User.Email != "admin@admin.com" || res.User.Role != user.RoleAdmin {
		t.Fatalf("Unexpected login response: %+v", res)
	}

	// 2. Test Invalid Password
	badBody, _ := json.Marshal(auth.LoginRequest{
		Email:    "admin@admin.com",
		Password: "wrongpassword",
	})
	reqBad := httptest.NewRequest("POST", "/api/auth/login", bytes.NewReader(badBody))
	reqBad.Header.Set("Content-Type", "application/json")
	wBad := httptest.NewRecorder()
	router.ServeHTTP(wBad, reqBad)

	if wBad.Code != http.StatusUnauthorized {
		t.Fatalf("Expected 401 Unauthorized, got %d", wBad.Code)
	}

	// 3. Test Create User via /api/users
	newUserBody, _ := json.Marshal(user.CreateUserRequest{
		Name:     "Rahul Sharma",
		Email:    "rahul@example.com",
		Password: "password123",
		Role:     "user",
		Phone:    "9123456780",
	})
	reqUser := httptest.NewRequest("POST", "/api/users", bytes.NewReader(newUserBody))
	reqUser.Header.Set("Content-Type", "application/json")
	wUser := httptest.NewRecorder()
	router.ServeHTTP(wUser, reqUser)

	if wUser.Code != http.StatusCreated {
		t.Fatalf("Expected 201 Created, got %d: %s", wUser.Code, wUser.Body.String())
	}

	// 4. Test List Users via /api/users
	reqList := httptest.NewRequest("GET", "/api/users", nil)
	wList := httptest.NewRecorder()
	router.ServeHTTP(wList, reqList)

	if wList.Code != http.StatusOK {
		t.Fatalf("Expected 200 OK, got %d", wList.Code)
	}

	// 5. Test Google OAuth Verify / User creation in DB
	googleUserPayload, _ := json.Marshal(map[string]interface{}{
		"mock_user": user.GoogleUserInfo{
			ID:      "google-user-12345",
			Email:   "aspirant.google@gmail.com",
			Name:    "Vikash Candidate",
			Picture: "https://lh3.googleusercontent.com/a/test-avatar",
		},
	})
	reqGoogle := httptest.NewRequest("POST", "/api/auth/google/verify", bytes.NewReader(googleUserPayload))
	reqGoogle.Header.Set("Content-Type", "application/json")
	wGoogle := httptest.NewRecorder()
	router.ServeHTTP(wGoogle, reqGoogle)

	if wGoogle.Code != http.StatusOK {
		t.Fatalf("Expected 200 OK for Google login, got %d: %s", wGoogle.Code, wGoogle.Body.String())
	}

	var gRes auth.LoginResponse
	if err := json.Unmarshal(wGoogle.Body.Bytes(), &gRes); err != nil {
		t.Fatalf("Failed to parse Google response: %v", err)
	}
	if !gRes.Success || gRes.User.Email != "aspirant.google@gmail.com" || gRes.User.AuthProvider != "google" {
		t.Fatalf("Unexpected Google user stored in DB: %+v", gRes.User)
	}

	// 6. Test Health Check endpoint
	reqHealth := httptest.NewRequest("GET", "/health", nil)
	wHealth := httptest.NewRecorder()
	router.ServeHTTP(wHealth, reqHealth)
	if wHealth.Code != http.StatusOK {
		t.Fatalf("Expected 200 OK for /health, got %d", wHealth.Code)
	}

	// 7. Test Jobs endpoint
	reqJobs := httptest.NewRequest("GET", "/api/jobs", nil)
	wJobs := httptest.NewRecorder()
	router.ServeHTTP(wJobs, reqJobs)
	if wJobs.Code != http.StatusOK {
		t.Fatalf("Expected 200 OK for /api/jobs, got %d", wJobs.Code)
	}

	// 8. Test RSS Feed endpoint
	reqRSS := httptest.NewRequest("GET", "/rss.xml", nil)
	wRSS := httptest.NewRecorder()
	router.ServeHTTP(wRSS, reqRSS)
	if wRSS.Code != http.StatusOK {
		t.Fatalf("Expected 200 OK for /rss.xml, got %d", wRSS.Code)
	}
}
