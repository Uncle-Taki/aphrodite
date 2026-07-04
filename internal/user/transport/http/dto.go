package http

import (
	"time"

	"github.com/google/uuid"

	"aphrodite/internal/user/domain"
)

type RegisterRequest struct {
	Username    string  `json:"username" binding:"required" example:"alice"`
	Email       string  `json:"email" binding:"required" example:"alice@example.com"`
	Password    string  `json:"password" binding:"required" example:"correct-horse-battery-staple"`
	PhoneNumber *string `json:"phone_number,omitempty" example:"+15551234567"`
	Role        string  `json:"role,omitempty" enums:"user,admin" example:"user"`
}

type AuthenticateRequest struct {
	Identifier string `json:"identifier" binding:"required" example:"alice"`
	Password   string `json:"password" binding:"required" example:"correct-horse-battery-staple"`
}

type UserResponse struct {
	ID          uuid.UUID `json:"id"`
	Username    string    `json:"username"`
	Email       string    `json:"email"`
	PhoneNumber *string   `json:"phone_number,omitempty"`
	Role        string    `json:"role" enums:"user,admin"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type AuthenticateResponse struct {
	Token string       `json:"token"`
	User  UserResponse `json:"user"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}

func toUserResponse(u *domain.User) UserResponse {
	return UserResponse{
		ID:          u.ID,
		Username:    u.Username,
		Email:       u.Email,
		PhoneNumber: u.PhoneNumber,
		Role:        string(u.Role),
		CreatedAt:   u.CreatedAt,
		UpdatedAt:   u.UpdatedAt,
	}
}
