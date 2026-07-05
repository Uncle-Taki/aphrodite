package domain

import (
	"strings"
	"time"

	"github.com/google/uuid"
)

type Role string

const (
	RoleUser  Role = "user"
	RoleAdmin Role = "admin"
)

func (r Role) Valid() bool {
	return r == RoleUser || r == RoleAdmin
}

type User struct {
	ID           uuid.UUID
	Username     string
	Email        string
	PasswordHash string
	PhoneNumber  *string
	Role         Role
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type UserDTO struct {
	ID           uuid.UUID
	Username     string
	Email        string
	PasswordHash string
	PhoneNumber  *string
	Role         Role
	Now          time.Time
}

type UpdateDTO struct {
	Username    string
	Email       string
	PhoneNumber *string
	Role        Role
	Now         time.Time
}

func NewUser(dto UserDTO) (*User, error) {
	username := strings.TrimSpace(dto.Username)
	email := strings.ToLower(strings.TrimSpace(dto.Email))

	if username == "" {
		return nil, ErrInvalidUsername
	}
	if !looksLikeEmail(email) {
		return nil, ErrInvalidEmail
	}
	if dto.PasswordHash == "" {
		return nil, ErrInvalidPasswordHash
	}
	if !dto.Role.Valid() {
		return nil, ErrInvalidRole
	}
	phone := dto.PhoneNumber
	if phone != nil {
		trimmed := strings.TrimSpace(*phone)
		if trimmed == "" {
			phone = nil
		} else {
			phone = &trimmed
		}
	}

	return &User{
		ID:           dto.ID,
		Username:     username,
		Email:        email,
		PasswordHash: dto.PasswordHash,
		PhoneNumber:  phone,
		Role:         dto.Role,
		CreatedAt:    dto.Now,
		UpdatedAt:    dto.Now,
	}, nil
}

func (u *User) Update(dto UpdateDTO) error {
	username := strings.TrimSpace(dto.Username)
	email := strings.ToLower(strings.TrimSpace(dto.Email))

	if username == "" {
		return ErrInvalidUsername
	}
	if !looksLikeEmail(email) {
		return ErrInvalidEmail
	}
	if !dto.Role.Valid() {
		return ErrInvalidRole
	}

	phone := dto.PhoneNumber
	if phone != nil {
		trimmed := strings.TrimSpace(*phone)
		if trimmed == "" {
			phone = nil
		} else {
			phone = &trimmed
		}
	}

	u.Username = username
	u.Email = email
	u.PhoneNumber = phone
	u.Role = dto.Role
	u.UpdatedAt = dto.Now
	return nil
}

func (u *User) ChangePassword(passwordHash string, now time.Time) error {
	if passwordHash == "" {
		return ErrInvalidPasswordHash
	}
	u.PasswordHash = passwordHash
	u.UpdatedAt = now
	return nil
}

func looksLikeEmail(s string) bool {
	at := strings.IndexByte(s, '@')
	if at <= 0 || at == len(s)-1 {
		return false
	}
	return strings.IndexByte(s[at+1:], '.') > 0
}
