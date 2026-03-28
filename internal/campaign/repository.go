package campaign

import (
	"context"
	"errors"
	"fmt"
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

// ── Campaigns ────────────────────────────────────────────────────────────────

func (r *Repository) CreateCampaign(ctx context.Context, name, description string, creatorID uuid.UUID) (*models.Campaign, error) {
	var c models.Campaign
	err := r.db.QueryRow(ctx, `
		INSERT INTO campaigns (name, description, creator_user_id)
		VALUES ($1, $2, $3)
		RETURNING id, name, description, creator_user_id, created_at, updated_at
	`, name, description, creatorID).Scan(&c.ID, &c.Name, &c.Description, &c.CreatorUserID, &c.CreatedAt, &c.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("create campaign: %w", err)
	}
	return &c, nil
}

func (r *Repository) GetCampaignByID(ctx context.Context, campaignID uuid.UUID) (*models.Campaign, error) {
	var c models.Campaign
	err := r.db.QueryRow(ctx, `
		SELECT id, name, description, creator_user_id, created_at, updated_at
		FROM campaigns WHERE id = $1
	`, campaignID).Scan(&c.ID, &c.Name, &c.Description, &c.CreatorUserID, &c.CreatedAt, &c.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrCampaignNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get campaign: %w", err)
	}
	return &c, nil
}

func (r *Repository) ListCampaignsByUser(ctx context.Context, userID uuid.UUID) ([]models.Campaign, error) {
	rows, err := r.db.Query(ctx, `
		SELECT c.id, c.name, c.description, c.creator_user_id, c.created_at, c.updated_at
		FROM campaigns c
		JOIN campaign_members cm ON cm.campaign_id = c.id
		WHERE cm.user_id = $1
		ORDER BY c.created_at DESC
	`, userID)
	if err != nil {
		return nil, fmt.Errorf("list campaigns: %w", err)
	}
	defer rows.Close()

	var campaigns []models.Campaign
	for rows.Next() {
		var c models.Campaign
		if err := rows.Scan(&c.ID, &c.Name, &c.Description, &c.CreatorUserID, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, err
		}
		campaigns = append(campaigns, c)
	}
	return campaigns, nil
}

func (r *Repository) UpdateCampaign(ctx context.Context, campaignID uuid.UUID, name, description string) (*models.Campaign, error) {
	var c models.Campaign
	err := r.db.QueryRow(ctx, `
		UPDATE campaigns
		SET name = $1, description = $2, updated_at = NOW()
		WHERE id = $3
		RETURNING id, name, description, creator_user_id, created_at, updated_at
	`, name, description, campaignID).Scan(&c.ID, &c.Name, &c.Description, &c.CreatorUserID, &c.CreatedAt, &c.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrCampaignNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("update campaign: %w", err)
	}
	return &c, nil
}

func (r *Repository) DeleteCampaign(ctx context.Context, campaignID uuid.UUID) error {
	tag, err := r.db.Exec(ctx, `DELETE FROM campaigns WHERE id = $1`, campaignID)
	if err != nil {
		return fmt.Errorf("delete campaign: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrCampaignNotFound
	}
	return nil
}

// ── Members ──────────────────────────────────────────────────────────────────

func (r *Repository) AddMember(ctx context.Context, campaignID, userID uuid.UUID, role string) (*models.CampaignMember, error) {
	var m models.CampaignMember
	err := r.db.QueryRow(ctx, `
		WITH inserted AS (
			INSERT INTO campaign_members (campaign_id, user_id, role)
			VALUES ($1, $2, $3)
			RETURNING id, campaign_id, user_id, role, joined_at
		)
		SELECT i.id, i.campaign_id, i.user_id, u.username, i.role, i.joined_at
		FROM inserted i
		JOIN users u ON u.id = i.user_id
	`, campaignID, userID, role).Scan(&m.ID, &m.CampaignID, &m.UserID, &m.Username, &m.Role, &m.JoinedAt)
	if err != nil {
		return nil, fmt.Errorf("add member: %w", err)
	}
	return &m, nil
}

func (r *Repository) GetMember(ctx context.Context, campaignID, userID uuid.UUID) (*models.CampaignMember, error) {
	var m models.CampaignMember
	err := r.db.QueryRow(ctx, `
		SELECT cm.id, cm.campaign_id, cm.user_id, u.username, cm.role, cm.joined_at
		FROM campaign_members cm
		JOIN users u ON u.id = cm.user_id
		WHERE cm.campaign_id = $1 AND cm.user_id = $2
	`, campaignID, userID).Scan(&m.ID, &m.CampaignID, &m.UserID, &m.Username, &m.Role, &m.JoinedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotMember
	}
	if err != nil {
		return nil, fmt.Errorf("get member: %w", err)
	}
	return &m, nil
}

func (r *Repository) ListMembers(ctx context.Context, campaignID uuid.UUID) ([]models.CampaignMember, error) {
	rows, err := r.db.Query(ctx, `
		SELECT cm.id, cm.campaign_id, cm.user_id, u.username, cm.role, cm.joined_at
		FROM campaign_members cm
		JOIN users u ON u.id = cm.user_id
		WHERE cm.campaign_id = $1
		ORDER BY cm.joined_at ASC
	`, campaignID)
	if err != nil {
		return nil, fmt.Errorf("list members: %w", err)
	}
	defer rows.Close()

	var members []models.CampaignMember
	for rows.Next() {
		var m models.CampaignMember
		if err := rows.Scan(&m.ID, &m.CampaignID, &m.UserID, &m.Username, &m.Role, &m.JoinedAt); err != nil {
			return nil, err
		}
		members = append(members, m)
	}
	return members, nil
}

func (r *Repository) CountGMs(ctx context.Context, campaignID uuid.UUID) (int, error) {
	var count int
	err := r.db.QueryRow(ctx, `
		SELECT COUNT(*) FROM campaign_members
		WHERE campaign_id = $1 AND role = 'gm'
	`, campaignID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count gms: %w", err)
	}
	return count, nil
}

func (r *Repository) UpdateMemberRole(ctx context.Context, campaignID, userID uuid.UUID, role string) error {
	tag, err := r.db.Exec(ctx, `
		UPDATE campaign_members SET role = $1
		WHERE campaign_id = $2 AND user_id = $3
	`, role, campaignID, userID)
	if err != nil {
		return fmt.Errorf("update member role: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotMember
	}
	return nil
}

func (r *Repository) RemoveMember(ctx context.Context, campaignID, userID uuid.UUID) error {
	tag, err := r.db.Exec(ctx, `
		DELETE FROM campaign_members
		WHERE campaign_id = $1 AND user_id = $2
	`, campaignID, userID)
	if err != nil {
		return fmt.Errorf("remove member: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotMember
	}
	return nil
}

// ── User lookup ──────────────────────────────────────────────────────────────

// FindUserByIdentifier busca um usuário por email ou username.
func (r *Repository) FindUserByIdentifier(ctx context.Context, identifier string) (uuid.UUID, string, error) {
	var userID uuid.UUID
	var username string
	err := r.db.QueryRow(ctx, `
		SELECT id, username FROM users
		WHERE email = $1 OR username = $1
		LIMIT 1
	`, identifier).Scan(&userID, &username)
	if errors.Is(err, pgx.ErrNoRows) {
		return uuid.Nil, "", ErrUserNotFound
	}
	if err != nil {
		return uuid.Nil, "", fmt.Errorf("find user: %w", err)
	}
	return userID, username, nil
}

// ── Invites ──────────────────────────────────────────────────────────────────

func (r *Repository) CreateInvite(ctx context.Context, campaignID, createdBy uuid.UUID, code string, expiresAt *time.Time) (*models.CampaignInvite, error) {
	var inv models.CampaignInvite
	err := r.db.QueryRow(ctx, `
		INSERT INTO campaign_invites (campaign_id, code, created_by, expires_at)
		VALUES ($1, $2, $3, $4)
		RETURNING id, campaign_id, code, created_by, expires_at, used_at, used_by, created_at
	`, campaignID, code, createdBy, expiresAt).Scan(
		&inv.ID, &inv.CampaignID, &inv.Code, &inv.CreatedBy,
		&inv.ExpiresAt, &inv.UsedAt, &inv.UsedBy, &inv.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("create invite: %w", err)
	}
	return &inv, nil
}

func (r *Repository) GetInviteByCode(ctx context.Context, code string) (*models.CampaignInvite, error) {
	var inv models.CampaignInvite
	err := r.db.QueryRow(ctx, `
		SELECT id, campaign_id, code, created_by, expires_at, used_at, used_by, created_at
		FROM campaign_invites WHERE code = $1
	`, code).Scan(
		&inv.ID, &inv.CampaignID, &inv.Code, &inv.CreatedBy,
		&inv.ExpiresAt, &inv.UsedAt, &inv.UsedBy, &inv.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrInviteNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get invite: %w", err)
	}
	return &inv, nil
}

func (r *Repository) MarkInviteUsed(ctx context.Context, inviteID, userID uuid.UUID) error {
	_, err := r.db.Exec(ctx, `
		UPDATE campaign_invites
		SET used_at = NOW(), used_by = $1
		WHERE id = $2
	`, userID, inviteID)
	return err
}
