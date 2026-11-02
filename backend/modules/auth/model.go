package auth

import "rojgarbridge-backend/modules/user"

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type LoginResponse struct {
	Success bool      `json:"success"`
	Token   string    `json:"token"`
	User    user.User `json:"user"`
	Message string    `json:"message,omitempty"`
}

type GoogleVerifyRequest struct {
	AccessToken string               `json:"access_token"`
	IDToken     string               `json:"id_token"`
	MockUser    *user.GoogleUserInfo `json:"mock_user,omitempty"`
}
