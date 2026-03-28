package campaign

import (
	"context"
	"crypto/rand"
	"encoding/base32"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/rubendubeux/inventory-manager/models"
)

var (
	ErrCampaignNotFound  = errors.New("campaign not found")
	ErrNotMember         = errors.New("user is not a member of this campaign")
	ErrAlreadyMember     = errors.New("user is already a member of this campaign")
	ErrLastGM            = errors.New("cannot remove or demote the last GM of a campaign")
	ErrForbidden         = errors.New("forbidden: insufficient role")
	ErrCannotModifyCreator = errors.New("cannot remove or demote the campaign creator")
	ErrUserNotFound      = errors.New("user not found")
	ErrInviteNotFound    = errors.New("invite not found or already used")
	ErrInviteExpired     = errors.New("invite link has expired")
)

type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

// ── Campaigns ────────────────────────────────────────────────────────────────

func (s *Service) CreateCampaign(ctx context.Context, creatorID uuid.UUID, name, description string) (*models.Campaign, error) {
	if name == "" {
		return nil, fmt.Errorf("campaign name is required")
	}

	campaign, err := s.repo.CreateCampaign(ctx, name, description, creatorID)
	if err != nil {
		return nil, err
	}

	if _, err := s.repo.AddMember(ctx, campaign.ID, creatorID, "gm"); err != nil {
		return nil, fmt.Errorf("add creator as gm: %w", err)
	}

	return campaign, nil
}

func (s *Service) GetCampaign(ctx context.Context, campaignID uuid.UUID) (*models.Campaign, error) {
	return s.repo.GetCampaignByID(ctx, campaignID)
}

func (s *Service) ListCampaigns(ctx context.Context, userID uuid.UUID) ([]models.Campaign, error) {
	return s.repo.ListCampaignsByUser(ctx, userID)
}

func (s *Service) UpdateCampaign(ctx context.Context, campaignID uuid.UUID, requesterRole, name, description string) (*models.Campaign, error) {
	if requesterRole != "gm" {
		return nil, ErrForbidden
	}
	if name == "" {
		return nil, fmt.Errorf("campaign name is required")
	}
	return s.repo.UpdateCampaign(ctx, campaignID, name, description)
}

func (s *Service) DeleteCampaign(ctx context.Context, campaignID uuid.UUID, requesterRole string) error {
	if requesterRole != "gm" {
		return ErrForbidden
	}
	return s.repo.DeleteCampaign(ctx, campaignID)
}

// ── Members ──────────────────────────────────────────────────────────────────

func (s *Service) ListMembers(ctx context.Context, campaignID uuid.UUID) ([]models.CampaignMember, error) {
	return s.repo.ListMembers(ctx, campaignID)
}

// AddMember adiciona um membro por user_id, email ou username.
func (s *Service) AddMember(ctx context.Context, campaignID uuid.UUID, requesterRole string, identifier string, role string) (*models.CampaignMember, error) {
	if requesterRole != "gm" {
		return nil, ErrForbidden
	}
	if role != "gm" && role != "player" {
		return nil, fmt.Errorf("invalid role: must be 'gm' or 'player'")
	}

	// Resolve identifier → user_id (tenta como UUID primeiro, depois email/username)
	targetUserID, err := s.resolveUser(ctx, identifier)
	if err != nil {
		return nil, err
	}

	member, err := s.repo.AddMember(ctx, campaignID, targetUserID, role)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return nil, ErrAlreadyMember
		}
		return nil, err
	}
	return member, nil
}

func (s *Service) UpdateMemberRole(ctx context.Context, campaignID uuid.UUID, requesterRole string, targetUserID uuid.UUID, newRole string) error {
	if requesterRole != "gm" {
		return ErrForbidden
	}
	if newRole != "gm" && newRole != "player" {
		return fmt.Errorf("invalid role: must be 'gm' or 'player'")
	}

	campaign, err := s.repo.GetCampaignByID(ctx, campaignID)
	if err != nil {
		return err
	}
	if targetUserID == campaign.CreatorUserID && newRole == "player" {
		return ErrCannotModifyCreator
	}

	if newRole == "player" {
		member, err := s.repo.GetMember(ctx, campaignID, targetUserID)
		if err != nil {
			return err
		}
		if member.Role == "gm" {
			count, err := s.repo.CountGMs(ctx, campaignID)
			if err != nil {
				return err
			}
			if count <= 1 {
				return ErrLastGM
			}
		}
	}

	return s.repo.UpdateMemberRole(ctx, campaignID, targetUserID, newRole)
}

func (s *Service) RemoveMember(ctx context.Context, campaignID uuid.UUID, requesterRole string, targetUserID uuid.UUID) error {
	if requesterRole != "gm" {
		return ErrForbidden
	}

	campaign, err := s.repo.GetCampaignByID(ctx, campaignID)
	if err != nil {
		return err
	}
	if targetUserID == campaign.CreatorUserID {
		return ErrCannotModifyCreator
	}

	member, err := s.repo.GetMember(ctx, campaignID, targetUserID)
	if err != nil {
		return err
	}
	if member.Role == "gm" {
		count, err := s.repo.CountGMs(ctx, campaignID)
		if err != nil {
			return err
		}
		if count <= 1 {
			return ErrLastGM
		}
	}

	return s.repo.RemoveMember(ctx, campaignID, targetUserID)
}

// ── Invites ──────────────────────────────────────────────────────────────────

// CreateInvite gera um link/código de convite para a campanha.
// expiresInHours = 0 → sem expiração.
func (s *Service) CreateInvite(ctx context.Context, campaignID uuid.UUID, requesterRole string, createdBy uuid.UUID, expiresInHours int) (*models.CampaignInvite, error) {
	if requesterRole != "gm" {
		return nil, ErrForbidden
	}

	code, err := generateCode()
	if err != nil {
		return nil, fmt.Errorf("generate invite code: %w", err)
	}

	var expiresAt *time.Time
	if expiresInHours > 0 {
		t := time.Now().Add(time.Duration(expiresInHours) * time.Hour)
		expiresAt = &t
	}

	return s.repo.CreateInvite(ctx, campaignID, createdBy, code, expiresAt)
}

// JoinByCode usa um código de convite para entrar na campanha como player.
func (s *Service) JoinByCode(ctx context.Context, code string, userID uuid.UUID) (*models.CampaignMember, error) {
	invite, err := s.repo.GetInviteByCode(ctx, code)
	if err != nil {
		return nil, ErrInviteNotFound
	}

	if invite.UsedAt != nil {
		return nil, ErrInviteNotFound
	}

	if invite.ExpiresAt != nil && time.Now().After(*invite.ExpiresAt) {
		return nil, ErrInviteExpired
	}

	member, err := s.repo.AddMember(ctx, invite.CampaignID, userID, "player")
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return nil, ErrAlreadyMember
		}
		return nil, err
	}

	_ = s.repo.MarkInviteUsed(ctx, invite.ID, userID)

	return member, nil
}

// ── Helpers ──────────────────────────────────────────────────────────────────

func (s *Service) resolveUser(ctx context.Context, identifier string) (uuid.UUID, error) {
	// Tenta como UUID direto
	if id, err := uuid.Parse(identifier); err == nil {
		return id, nil
	}
	// Busca por email ou username
	userID, _, err := s.repo.FindUserByIdentifier(ctx, strings.TrimSpace(identifier))
	return userID, err
}

func generateCode() (string, error) {
	b := make([]byte, 5) // 5 bytes → 8 chars em base32
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return strings.TrimRight(base32.StdEncoding.EncodeToString(b), "="), nil
}
