package character

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/rubendubeux/inventory-manager/models"
)

type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) CreateCharacter(ctx context.Context, campaignID, requesterID uuid.UUID, requesterRole string, ownerUserID uuid.UUID, name, description string, maxCarryWeightKg *float64) (*models.Character, error) {
	if name == "" {
		return nil, fmt.Errorf("name is required")
	}

	// Players can only create characters for themselves
	if requesterRole == "player" {
		ownerUserID = requesterID
	}

	// Verify owner is a campaign member
	isMember, err := s.repo.IsMember(ctx, campaignID, ownerUserID)
	if err != nil {
		return nil, err
	}
	if !isMember {
		return nil, ErrOwnerNotMember
	}

	return s.repo.CreateCharacter(ctx, campaignID, ownerUserID, name, description, maxCarryWeightKg)
}

func (s *Service) GetCharacter(ctx context.Context, characterID, requesterID uuid.UUID, requesterRole string) (*models.Character, error) {
	ch, err := s.repo.GetCharacterByID(ctx, characterID)
	if err != nil {
		return nil, err
	}

	if requesterRole == "player" && ch.OwnerUserID != requesterID {
		return nil, ErrForbidden
	}

	return ch, nil
}

func (s *Service) ListCharacters(ctx context.Context, campaignID, requesterID uuid.UUID, requesterRole string) ([]models.Character, error) {
	if requesterRole == "gm" {
		return s.repo.ListCharactersByCampaign(ctx, campaignID)
	}
	return s.repo.ListCharactersByOwner(ctx, campaignID, requesterID)
}

func (s *Service) UpdateCharacter(ctx context.Context, characterID, requesterID uuid.UUID, requesterRole, name, description string, maxCarryWeightKg *float64) (*models.Character, error) {
	if name == "" {
		return nil, fmt.Errorf("name is required")
	}

	ch, err := s.repo.GetCharacterByID(ctx, characterID)
	if err != nil {
		return nil, err
	}

	if requesterRole == "player" && ch.OwnerUserID != requesterID {
		return nil, ErrForbidden
	}

	return s.repo.UpdateCharacter(ctx, characterID, name, description, maxCarryWeightKg)
}

func (s *Service) DeleteCharacter(ctx context.Context, characterID, requesterID uuid.UUID, requesterRole string) error {
	ch, err := s.repo.GetCharacterByID(ctx, characterID)
	if err != nil {
		return err
	}

	if requesterRole == "player" && ch.OwnerUserID != requesterID {
		return ErrForbidden
	}

	return s.repo.DeleteCharacter(ctx, characterID)
}
