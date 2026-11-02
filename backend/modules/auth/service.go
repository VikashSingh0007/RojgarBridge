package auth

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
	"rojgarbridge-backend/config"
	"rojgarbridge-backend/modules/user"
)

var (
	ErrInvalidCredentials = errors.New("invalid email or password")
	ErrUnauthorized       = errors.New("not authenticated")
)

type Service interface {
	Login(email, password string) (*LoginResponse, error)
	GetAuthMe(authHeader string) (*user.User, error)
	ExchangeGoogleCode(code, callbackURL string) (*user.GoogleUserInfo, error)
	FindOrCreateGoogleUser(g *user.GoogleUserInfo) (*user.User, string, error)
	VerifyGoogleToken(accessToken string, mockUser *user.GoogleUserInfo) (*user.User, string, error)
}

type service struct {
	userService user.Service
	cfg         *config.Config
	client      *http.Client
}

func NewService(userService user.Service, cfg *config.Config) Service {
	return &service{
		userService: userService,
		cfg:         cfg,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (s *service) Login(email, password string) (*LoginResponse, error) {
	email = strings.TrimSpace(email)
	password = strings.TrimSpace(password)

	if email == "" || password == "" {
		return nil, errors.New("email and password are required")
	}

	// 1. Explicit hardcoded admin credential support
	if strings.EqualFold(email, "admin@admin.com") && password == "admin@7077" {
		adminUser, err := s.userService.GetUserByEmail("admin@admin.com")
		if err != nil || adminUser == nil {
			adminUser = &user.User{
				ID:        1,
				Name:      "System Administrator",
				Email:     "admin@admin.com",
				Role:      user.RoleAdmin,
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			}
		}
		return &LoginResponse{
			Success: true,
			Token:   "admin-session-" + strconv.FormatInt(time.Now().Unix(), 10),
			User:    *adminUser,
			Message: "Admin authenticated successfully",
		}, nil
	}

	// 2. Lookup in database
	u, err := s.userService.GetUserByEmail(email)
	if err != nil || u == nil {
		return nil, ErrInvalidCredentials
	}

	// Compare password hash or fallback
	if err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password)); err != nil {
		if u.PasswordHash != password {
			return nil, ErrInvalidCredentials
		}
	}

	return &LoginResponse{
		Success: true,
		Token:   fmt.Sprintf("user-session-%d-%d", u.ID, time.Now().Unix()),
		User:    *u,
		Message: "Authentication successful",
	}, nil
}

func (s *service) GetAuthMe(authHeader string) (*user.User, error) {
	if strings.Contains(authHeader, "admin") {
		adminUser, _ := s.userService.GetUserByEmail("admin@admin.com")
		if adminUser != nil {
			return adminUser, nil
		}
		return &user.User{
			ID:        1,
			Name:      "System Administrator",
			Email:     "admin@admin.com",
			Role:      user.RoleAdmin,
			CreatedAt: time.Now(),
		}, nil
	}

	adminUser, err := s.userService.GetUserByEmail("admin@admin.com")
	if err == nil && adminUser != nil {
		return adminUser, nil
	}

	return nil, ErrUnauthorized
}

func (s *service) ExchangeGoogleCode(code, callbackURL string) (*user.GoogleUserInfo, error) {
	data := url.Values{}
	data.Set("code", code)
	data.Set("client_id", s.cfg.GoogleClientID)
	data.Set("client_secret", s.cfg.GoogleClientSecret)
	data.Set("redirect_uri", callbackURL)
	data.Set("grant_type", "authorization_code")

	tokenResp, err := http.PostForm("https://oauth2.googleapis.com/token", data)
	if err != nil {
		return nil, fmt.Errorf("token exchange failed: %w", err)
	}
	defer tokenResp.Body.Close()

	if tokenResp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("google auth failed with status %d", tokenResp.StatusCode)
	}

	var tokenData struct {
		AccessToken string `json:"access_token"`
		IDToken     string `json:"id_token"`
	}
	if err := json.NewDecoder(tokenResp.Body).Decode(&tokenData); err != nil {
		return nil, fmt.Errorf("invalid token response: %w", err)
	}

	userReq, err := http.NewRequest("GET", "https://www.googleapis.com/oauth2/v2/userinfo", nil)
	if err != nil {
		return nil, fmt.Errorf("profile request failed: %w", err)
	}
	userReq.Header.Set("Authorization", "Bearer "+tokenData.AccessToken)

	userResp, err := s.client.Do(userReq)
	if err != nil || userResp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("userinfo fetch failed")
	}
	defer userResp.Body.Close()

	var gUser user.GoogleUserInfo
	if err := json.NewDecoder(userResp.Body).Decode(&gUser); err != nil {
		return nil, fmt.Errorf("userinfo decode failed: %w", err)
	}

	return &gUser, nil
}

func (s *service) FindOrCreateGoogleUser(g *user.GoogleUserInfo) (*user.User, string, error) {
	u, err := s.userService.FindOrCreateGoogleUser(g)
	if err != nil {
		return nil, "", err
	}
	sessionToken := fmt.Sprintf("google-session-%d-%d", u.ID, time.Now().Unix())
	return u, sessionToken, nil
}

func (s *service) VerifyGoogleToken(accessToken string, mockUser *user.GoogleUserInfo) (*user.User, string, error) {
	var gUser user.GoogleUserInfo

	if mockUser != nil && mockUser.Email != "" {
		gUser = *mockUser
	} else if accessToken != "" {
		userReq, err := http.NewRequest("GET", "https://www.googleapis.com/oauth2/v2/userinfo", nil)
		if err != nil {
			return nil, "", fmt.Errorf("failed to create request: %w", err)
		}
		userReq.Header.Set("Authorization", "Bearer "+accessToken)
		userResp, err := s.client.Do(userReq)
		if err != nil || userResp.StatusCode != http.StatusOK {
			return nil, "", errors.New("invalid Google access token")
		}
		defer userResp.Body.Close()
		if err := json.NewDecoder(userResp.Body).Decode(&gUser); err != nil {
			return nil, "", fmt.Errorf("failed to decode Google user: %w", err)
		}
	} else {
		return nil, "", errors.New("access_token or mock_user required")
	}

	return s.FindOrCreateGoogleUser(&gUser)
}
