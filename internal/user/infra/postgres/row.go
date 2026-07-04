package postgres

import (
	"time"

	"github.com/google/uuid"

	"aphrodite/internal/user/domain"
)

type userRow struct {
	ID           uuid.UUID `gorm:"type:uuid;primaryKey"`
	Username     string    `gorm:"type:varchar(64);uniqueIndex;not null"`
	Email        string    `gorm:"type:varchar(320);uniqueIndex;not null"`
	PasswordHash string    `gorm:"type:varchar(255);not null"`
	PhoneNumber  *string   `gorm:"type:varchar(32)"`
	Role         string    `gorm:"type:varchar(16);not null;default:'user'"`
	CreatedAt    time.Time `gorm:"not null"`
	UpdatedAt    time.Time `gorm:"not null"`
}

func (userRow) TableName() string { return "users" }

func fromDomain(u *domain.User) userRow {
	return userRow{
		ID:           u.ID,
		Username:     u.Username,
		Email:        u.Email,
		PasswordHash: u.PasswordHash,
		PhoneNumber:  u.PhoneNumber,
		Role:         string(u.Role),
		CreatedAt:    u.CreatedAt,
		UpdatedAt:    u.UpdatedAt,
	}
}

func (r userRow) toDomain() *domain.User {
	return &domain.User{
		ID:           r.ID,
		Username:     r.Username,
		Email:        r.Email,
		PasswordHash: r.PasswordHash,
		PhoneNumber:  r.PhoneNumber,
		Role:         domain.Role(r.Role),
		CreatedAt:    r.CreatedAt,
		UpdatedAt:    r.UpdatedAt,
	}
}
