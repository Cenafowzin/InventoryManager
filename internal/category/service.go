package category

import (
	"context"
	"errors"
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

func (s *Service) CreateCategory(ctx context.Context, campaignID uuid.UUID, requesterRole, name, color string) (*models.Category, error) {
	if requesterRole != "gm" {
		return nil, errors.New("forbidden")
	}
	if name == "" {
		return nil, fmt.Errorf("name is required")
	}
	return s.repo.CreateCategory(ctx, campaignID, name, color)
}

func (s *Service) ListCategories(ctx context.Context, campaignID uuid.UUID) ([]models.Category, error) {
	return s.repo.ListCategories(ctx, campaignID)
}

func (s *Service) UpdateCategory(ctx context.Context, id uuid.UUID, requesterRole, name, color string) (*models.Category, error) {
	if requesterRole != "gm" {
		return nil, errors.New("forbidden")
	}
	if name == "" {
		return nil, fmt.Errorf("name is required")
	}
	return s.repo.UpdateCategory(ctx, id, name, color)
}

func (s *Service) DeleteCategory(ctx context.Context, id uuid.UUID, requesterRole string) error {
	if requesterRole != "gm" {
		return errors.New("forbidden")
	}
	return s.repo.DeleteCategory(ctx, id)
}

func (s *Service) ValidateCategoryIDs(ctx context.Context, campaignID uuid.UUID, categoryIDs []uuid.UUID) error {
	return s.repo.ValidateCategoryIDs(ctx, campaignID, categoryIDs)
}

func (s *Service) SetShopItemCategories(ctx context.Context, shopItemID uuid.UUID, categoryIDs []uuid.UUID) error {
	return s.repo.SetShopItemCategories(ctx, shopItemID, categoryIDs)
}
