package inventory

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/rubendubeux/inventory-manager/internal/character"
	"github.com/rubendubeux/inventory-manager/models"
)

type StorageService struct {
	repo     *StorageRepository
	charRepo *character.Repository
}

func NewStorageService(repo *StorageRepository, charRepo *character.Repository) *StorageService {
	return &StorageService{repo: repo, charRepo: charRepo}
}

// EnsureDefaultSpace cria o espaço "Corpo" para um personagem recém-criado.
func (s *StorageService) EnsureDefaultSpace(ctx context.Context, characterID uuid.UUID) error {
	_, err := s.repo.CreateDefaultSpace(ctx, characterID)
	return err
}

func (s *StorageService) CreateStorageSpace(ctx context.Context, characterID, requesterID uuid.UUID, requesterRole, name, description string, countsTowardLoad bool, capacityKg *float64, itemID *uuid.UUID) (*models.StorageSpace, error) {
	if err := s.checkAccess(ctx, characterID, requesterID, requesterRole); err != nil {
		return nil, err
	}
	if name == "" {
		return nil, fmt.Errorf("name is required")
	}
	return s.repo.CreateStorageSpace(ctx, characterID, name, description, countsTowardLoad, capacityKg, itemID)
}

func (s *StorageService) ListStorageSpaces(ctx context.Context, characterID, requesterID uuid.UUID, requesterRole string) ([]models.StorageSpace, error) {
	if err := s.checkAccess(ctx, characterID, requesterID, requesterRole); err != nil {
		return nil, err
	}
	return s.repo.ListStorageSpaces(ctx, characterID)
}

func (s *StorageService) UpdateStorageSpace(ctx context.Context, spaceID, requesterID uuid.UUID, requesterRole, name, description string, countsTowardLoad bool, capacityKg *float64) (*models.StorageSpace, error) {
	ss, err := s.repo.GetStorageSpaceByID(ctx, spaceID)
	if err != nil {
		return nil, err
	}
	if err := s.checkAccess(ctx, ss.CharacterID, requesterID, requesterRole); err != nil {
		return nil, err
	}
	if name == "" {
		return nil, fmt.Errorf("name is required")
	}
	return s.repo.UpdateStorageSpace(ctx, spaceID, name, description, countsTowardLoad, capacityKg)
}

func (s *StorageService) DeleteStorageSpace(ctx context.Context, spaceID, requesterID uuid.UUID, requesterRole string) error {
	ss, err := s.repo.GetStorageSpaceByID(ctx, spaceID)
	if err != nil {
		return err
	}
	if err := s.checkAccess(ctx, ss.CharacterID, requesterID, requesterRole); err != nil {
		return err
	}
	if ss.IsDefault {
		return ErrCannotDeleteDefault
	}
	defaultSpace, err := s.repo.GetDefaultStorageSpace(ctx, ss.CharacterID)
	if err != nil {
		return fmt.Errorf("get default space: %w", err)
	}
	if err := s.repo.ReassignItemsToDefault(ctx, spaceID, defaultSpace.ID); err != nil {
		return fmt.Errorf("reassign items: %w", err)
	}
	return s.repo.DeleteStorageSpace(ctx, spaceID)
}

func (s *StorageService) checkAccess(ctx context.Context, characterID, requesterID uuid.UUID, requesterRole string) error {
	if requesterRole == "gm" {
		return nil
	}
	ch, err := s.charRepo.GetCharacterByID(ctx, characterID)
	if err != nil {
		return err
	}
	if ch.OwnerUserID != requesterID {
		return ErrForbidden
	}
	return nil
}
