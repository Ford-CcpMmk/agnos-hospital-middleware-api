package dto

import "time"

type CreateStaffRequest struct {
	Username string `json:"username" binding:"required,min=3,max=100"`
	Password string `json:"password" binding:"required,min=8,max=72"`
	Hospital string `json:"hospital" binding:"required,max=50"`
}

type CreateStaffResponse struct {
	Data CreateStaffData `json:"data"`
}

type CreateStaffData struct {
	ID        int64     `json:"id"`
	Username  string    `json:"username"`
	Hospital  string    `json:"hospital"`
	CreatedAt time.Time `json:"created_at"`
}

type LoginStaffRequest struct {
	Username string `json:"username" binding:"required,max=100"`
	Password string `json:"password" binding:"required,max=72"`
	Hospital string `json:"hospital" binding:"required,max=50"`
}

type LoginStaffResponse struct {
	Data LoginStaffData `json:"data"`
}

type LoginStaffData struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int64  `json:"expires_in"`
}
