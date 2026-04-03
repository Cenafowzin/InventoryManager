package discord

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"

	"github.com/rubendubeux/inventory-manager/models"
)

const linkCodeTTL = 10 * time.Minute

type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

// GenerateLinkCode cria um código temporário para o usuário iniciar a vinculação pelo Discord.
func (s *Service) GenerateLinkCode(ctx context.Context, userID uuid.UUID) (string, error) {
	return s.repo.CreateLinkCode(ctx, userID, linkCodeTTL)
}

// ConfirmLink finaliza a vinculação: valida o código e salva discord_id → user_id.
func (s *Service) ConfirmLink(ctx context.Context, code, discordID, discordTag string) error {
	userID, err := s.repo.ConsumeLinkCode(ctx, code)
	if err != nil {
		return err
	}
	return s.repo.CreateLink(ctx, userID, discordID, discordTag)
}

// GetUserByDiscordID retorna o usuário do site associado ao discord_id.
func (s *Service) GetUserByDiscordID(ctx context.Context, discordID string) (*models.User, error) {
	return s.repo.GetLinkByDiscordID(ctx, discordID)
}

// IsLinked verifica se o usuário já tem Discord vinculado.
func (s *Service) IsLinked(ctx context.Context, userID uuid.UUID) (bool, error) {
	_, err := s.repo.GetLinkByUserID(ctx, userID)
	if errors.Is(err, ErrNotLinked) {
		return false, nil
	}
	return err == nil, err
}

// Unlink remove a vinculação de um usuário.
func (s *Service) Unlink(ctx context.Context, userID uuid.UUID) error {
	return s.repo.DeleteLink(ctx, userID)
}
