package api_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"rojgarbridge-backend/internal/api"
	"rojgarbridge-backend/internal/config"
	"rojgarbridge-backend/internal/db"
	"rojgarbridge-backend/internal/models"
)

func setupTestApp(t *testing.T) (*chiRouterWrapper, *db.DB) {
	tempDB := filepath.Join(t.TempDir(), "api_test.db")
	database, err := db.InitDB(tempDB, "")
	if err != nil {
		t.Fatalf("InitDB failed: %v", err)
	}

	cfg := &config.Config{
		FrontendURL:       "http://localhost:3000",
		GoogleRedirectURL: "http://localhost:8080/api/auth/google/callback",
	}
	handler := api.NewHandler(database, nil, nil, cfg)
	router := api.NewRouter(handler)

	return &chiRouterWrapper{router: router}, database
}

type chiRouterWrapper struct {
	router http.Handler
}

func (w *chiRouterWrapper) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	w.router.ServeHTTP(rw, req)
}

func TestAuthLoginEndpoint(t *testing.T) {
	app, dbConn := setupTestApp(t)
	defer dbConn.Close()

	// 1. Test Admin Login with hardcoded credentials
	body, _ := json.Marshal(models.LoginRequest{
		Email:    "admin@admin.com",
		Password: "admin@7077",
	})
	req := httptest.NewRequest("POST", "/api/auth/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	app.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200 OK, got %d: %s", w.Code, w.Body.String())
	}

	var res models.LoginResponse
	if err := json.Unmarshal(w.Body.Bytes(), &res); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}
	if !res.Success || res.User.Email != "admin@admin.com" || res.User.Role != models.RoleAdmin {
		t.Fatalf("Unexpected login response: %+v", res)
	}

	// 2. Test Invalid Password
	badBody, _ := json.Marshal(models.LoginRequest{
		Email:    "admin@admin.com",
		Password: "wrongpassword",
	})
	reqBad := httptest.NewRequest("POST", "/api/auth/login", bytes.NewReader(badBody))
	reqBad.Header.Set("Content-Type", "application/json")
	wBad := httptest.NewRecorder()
	app.ServeHTTP(wBad, reqBad)

	if wBad.Code != http.StatusUnauthorized {
		t.Fatalf("Expected 401 Unauthorized, got %d", wBad.Code)
	}

	// 3. Test Create User via /api/users
	newUserBody, _ := json.Marshal(models.CreateUserRequest{
		Name:     "Rahul Sharma",
		Email:    "rahul@example.com",
		Password: "password123",
		Role:     "user",
		Phone:    "9123456780",
	})
	reqUser := httptest.NewRequest("POST", "/api/users", bytes.NewReader(newUserBody))
	reqUser.Header.Set("Content-Type", "application/json")
	wUser := httptest.NewRecorder()
	app.ServeHTTP(wUser, reqUser)

	if wUser.Code != http.StatusCreated {
		t.Fatalf("Expected 201 Created, got %d: %s", wUser.Code, wUser.Body.String())
	}

	// 4. Test List Users via /api/users
	reqList := httptest.NewRequest("GET", "/api/users", nil)
	wList := httptest.NewRecorder()
	app.ServeHTTP(wList, reqList)

	if wList.Code != http.StatusOK {
		t.Fatalf("Expected 200 OK, got %d", wList.Code)
	}

	// 5. Test Google OAuth Verify / User creation in DB
	googleUserPayload, _ := json.Marshal(map[string]interface{}{
		"mock_user": models.GoogleUserInfo{
			ID:      "google-user-12345",
			Email:   "aspirant.google@gmail.com",
			Name:    "Vikash Candidate",
			Picture: "https://lh3.googleusercontent.com/a/test-avatar",
		},
	})
	reqGoogle := httptest.NewRequest("POST", "/api/auth/google/verify", bytes.NewReader(googleUserPayload))
	reqGoogle.Header.Set("Content-Type", "application/json")
	wGoogle := httptest.NewRecorder()
	app.ServeHTTP(wGoogle, reqGoogle)

	if wGoogle.Code != http.StatusOK {
		t.Fatalf("Expected 200 OK for Google login, got %d: %s", wGoogle.Code, wGoogle.Body.String())
	}

	var gRes models.LoginResponse
	if err := json.Unmarshal(wGoogle.Body.Bytes(), &gRes); err != nil {
		t.Fatalf("Failed to parse Google response: %v", err)
	}
	if !gRes.Success || gRes.User.Email != "aspirant.google@gmail.com" || gRes.User.AuthProvider != "google" {
		t.Fatalf("Unexpected Google user stored in DB: %+v", gRes.User)
	}
}
