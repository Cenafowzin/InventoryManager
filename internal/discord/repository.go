package discord

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/rubendubeux/inventory-manager/models"
)

type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

// ── Link codes ────────────────────────────────────────────────────────────────

func (r *Repository) CreateLinkCode(ctx context.Context, userID uuid.UUID, ttl time.Duration) (string, error) {
	b := make([]byte, 3)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	code := hex.EncodeToString(b) // 6 hex chars

	_, err := r.db.Exec(ctx, `
		INSERT INTO discord_link_codes (code, user_id, expires_at)
		VALUES ($1, $2, $3)
		ON CONFLICT (code) DO UPDATE SET user_id = $2, expires_at = $3
	`, code, userID, time.Now().Add(ttl))
	if err != nil {
		return "", err
	}
	return code, nil
}

func (r *Repository) ConsumeLinkCode(ctx context.Context, code string) (uuid.UUID, error) {
	var userID uuid.UUID
	var expiresAt time.Time
	err := r.db.QueryRow(ctx, `
		DELETE FROM discord_link_codes WHERE code = $1
		RETURNING user_id, expires_at
	`, code).Scan(&userID, &expiresAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return uuid.Nil, ErrInvalidCode
	}
	if err != nil {
		return uuid.Nil, err
	}
	if time.Now().After(expiresAt) {
		return uuid.Nil, ErrCodeExpired
	}
	return userID, nil
}

// ── Discord links ─────────────────────────────────────────────────────────────

func (r *Repository) CreateLink(ctx context.Context, userID uuid.UUID, discordID, discordTag string) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO discord_links (user_id, discord_id, discord_tag)
		VALUES ($1, $2, $3)
		ON CONFLICT (user_id) DO UPDATE SET discord_id = $2, discord_tag = $3, linked_at = NOW()
	`, userID, discordID, discordTag)
	return err
}

func (r *Repository) GetLinkByDiscordID(ctx context.Context, discordID string) (*models.User, error) {
	var u models.User
	err := r.db.QueryRow(ctx, `
		SELECT u.id, u.username, u.email, u.password_hash, u.created_at
		FROM users u
		JOIN discord_links dl ON dl.user_id = u.id
		WHERE dl.discord_id = $1
	`, discordID).Scan(&u.ID, &u.Username, &u.Email, &u.PasswordHash, &u.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotLinked
	}
	return &u, err
}

func (r *Repository) GetLinkByUserID(ctx context.Context, userID uuid.UUID) (discordID string, err error) {
	err = r.db.QueryRow(ctx, `
		SELECT discord_id FROM discord_links WHERE user_id = $1
	`, userID).Scan(&discordID)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrNotLinked
	}
	return discordID, err
}

func (r *Repository) DeleteLink(ctx context.Context, userID uuid.UUID) error {
	_, err := r.db.Exec(ctx, `DELETE FROM discord_links WHERE user_id = $1`, userID)
	return err
}
