package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"aphrodite/internal/user/domain"
	"aphrodite/internal/user/usecase"
)

type sessionRow struct {
	ID         uuid.UUID  `gorm:"type:uuid;primaryKey"`
	UserID     uuid.UUID  `gorm:"type:uuid;index;not null"`
	User       userRef    `gorm:"foreignKey:UserID;references:ID;constraint:OnDelete:CASCADE"`
	TokenHash  string     `gorm:"type:char(64);uniqueIndex;not null"`
	ExpiresAt  time.Time  `gorm:"index;not null"`
	RevokedAt  *time.Time `gorm:"index"`
	CreatedAt  time.Time  `gorm:"not null"`
	LastSeenAt time.Time  `gorm:"not null"`
}

type userRef struct {
	ID uuid.UUID `gorm:"type:uuid;primaryKey"`
}

func (userRef) TableName() string { return "users" }

func (sessionRow) TableName() string { return "cms_sessions" }

type SessionRepository struct{ db *gorm.DB }

var _ usecase.SessionRepository = (*SessionRepository)(nil)

func NewSessionRepository(db *gorm.DB) *SessionRepository { return &SessionRepository{db: db} }

func (r *SessionRepository) Create(ctx context.Context, s *domain.Session) error {
	return r.db.WithContext(ctx).Create(&sessionRow{ID: s.ID, UserID: s.UserID, TokenHash: s.TokenHash, ExpiresAt: s.ExpiresAt, CreatedAt: s.CreatedAt, LastSeenAt: s.LastSeenAt}).Error
}

func (r *SessionRepository) FindActiveByHash(ctx context.Context, hash string, now time.Time) (*domain.Session, error) {
	var row sessionRow
	err := r.db.WithContext(ctx).Where("token_hash = ? AND revoked_at IS NULL AND expires_at > ?", hash, now).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &domain.Session{ID: row.ID, UserID: row.UserID, TokenHash: row.TokenHash, ExpiresAt: row.ExpiresAt, RevokedAt: row.RevokedAt, CreatedAt: row.CreatedAt, LastSeenAt: row.LastSeenAt}, nil
}

func (r *SessionRepository) Touch(ctx context.Context, id uuid.UUID, now time.Time) error {
	return r.db.WithContext(ctx).Model(&sessionRow{}).Where("id = ?", id).Update("last_seen_at", now).Error
}

func (r *SessionRepository) Revoke(ctx context.Context, id uuid.UUID, now time.Time) error {
	return r.db.WithContext(ctx).Model(&sessionRow{}).Where("id = ? AND revoked_at IS NULL", id).Update("revoked_at", now).Error
}
