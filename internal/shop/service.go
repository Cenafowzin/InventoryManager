package shop

import (
	"context"

	"github.com/google/uuid"

	"github.com/rubendubeux/inventory-manager/models"
)

// Dependencies interfaces
type CategoryService interface {
	ValidateCategoryIDs(ctx context.Context, campaignID uuid.UUID, categoryIDs []uuid.UUID) error
	SetShopItemCategories(ctx context.Context, shopItemID uuid.UUID, categoryIDs []uuid.UUID) error
}

type CoinService interface {
	GetDefaultCoin(ctx context.Context, campaignID uuid.UUID) (*models.CoinType, error)
	GetCoinByID(ctx context.Context, id uuid.UUID) (*models.CoinType, error)
}

// Service struct
type Service struct {
	repo        *Repository
	categorySvc CategoryService
	coinSvc     CoinService
}

func NewService(repo *Repository, categorySvc CategoryService, coinSvc CoinService) *Service {
	return &Service{
		repo:        repo,
		categorySvc: categorySvc,
		coinSvc:     coinSvc,
	}
}

type CreateShopItemInput struct {
	Name        string
	Description string
	Emoji       string
	WeightKg    float64
	BaseValue   float64
	ValueCoinID *uuid.UUID
	IsAvailable bool
	CategoryIDs []uuid.UUID
}

type UpdateShopItemInput struct {
	Name        string
	Description string
	Emoji       string
	WeightKg    float64
	BaseValue   float64
	ValueCoinID *uuid.UUID
	IsAvailable *bool
	CategoryIDs []uuid.UUID
}

type ListShopItemsFilters struct {
	CategoryID *uuid.UUID
}

func (s *Service) CreateShopItem(
	ctx context.Context,
	campaignID uuid.UUID,
	requesterRole string,
	input CreateShopItemInput,
) (*models.ShopItem, error) {
	if requesterRole != "gm" {
		return nil, ErrForbidden
	}

	valueCoinID := input.ValueCoinID
	if valueCoinID == nil {
		defaultCoin, err := s.coinSvc.GetDefaultCoin(ctx, campaignID)
		if err != nil {
			return nil, err
		}
		valueCoinID = &defaultCoin.ID
	} else {
		coin, err := s.coinSvc.GetCoinByID(ctx, *valueCoinID)
		if err != nil {
			return nil, err
		}
		if coin.CampaignID != campaignID {
			return nil, ErrCoinNotInCampaign
		}
	}

	if len(input.CategoryIDs) > 0 {
		if err := s.categorySvc.ValidateCategoryIDs(ctx, campaignID, input.CategoryIDs); err != nil {
			return nil, err
		}
	}

	item, err := s.repo.CreateShopItem(
		ctx, campaignID, valueCoinID,
		input.Name, input.Description, input.Emoji,
		input.WeightKg, input.BaseValue, input.IsAvailable,
	)
	if err != nil {
		return nil, err
	}

	if len(input.CategoryIDs) > 0 {
		if err := s.categorySvc.SetShopItemCategories(ctx, item.ID, input.CategoryIDs); err != nil {
			return nil, err
		}
		item, err = s.repo.GetShopItemByID(ctx, item.ID)
		if err != nil {
			return nil, err
		}
	}

	return item, nil
}

func (s *Service) ListShopItems(
	ctx context.Context,
	campaignID uuid.UUID,
	requesterRole string,
	filters ListShopItemsFilters,
	includeUnavailable bool,
) ([]models.ShopItem, error) {
	onlyAvailable := true
	if requesterRole == "gm" && includeUnavailable {
		onlyAvailable = false
	}

	return s.repo.ListShopItems(ctx, campaignID, ShopFilters{
		OnlyAvailable: onlyAvailable,
		CategoryID:    filters.CategoryID,
	})
}

func (s *Service) GetShopItemByID(
	ctx context.Context,
	id uuid.UUID,
	requesterRole string,
) (*models.ShopItem, error) {
	item, err := s.repo.GetShopItemByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if requesterRole != "gm" && !item.IsAvailable {
		return nil, ErrShopItemNotFound
	}
	return item, nil
}

func (s *Service) UpdateShopItem(
	ctx context.Context,
	id uuid.UUID,
	campaignID uuid.UUID,
	requesterRole string,
	input UpdateShopItemInput,
) (*models.ShopItem, error) {
	if requesterRole != "gm" {
		return nil, ErrForbidden
	}

	valueCoinID := input.ValueCoinID
	if valueCoinID != nil {
		coin, err := s.coinSvc.GetCoinByID(ctx, *valueCoinID)
		if err != nil {
			return nil, err
		}
		if coin.CampaignID != campaignID {
			return nil, ErrCoinNotInCampaign
		}
	}

	item, err := s.repo.UpdateShopItem(
		ctx, id, valueCoinID,
		input.Name, input.Description, input.Emoji,
		input.WeightKg, input.BaseValue, input.IsAvailable,
	)
	if err != nil {
		return nil, err
	}

	if len(input.CategoryIDs) > 0 {
		if err := s.categorySvc.ValidateCategoryIDs(ctx, campaignID, input.CategoryIDs); err != nil {
			return nil, err
		}
		if err := s.categorySvc.SetShopItemCategories(ctx, id, input.CategoryIDs); err != nil {
			return nil, err
		}
		item, err = s.repo.GetShopItemByID(ctx, id)
		if err != nil {
			return nil, err
		}
	}

	return item, nil
}

func (s *Service) DeleteShopItem(
	ctx context.Context,
	id uuid.UUID,
	requesterRole string,
) error {
	if requesterRole != "gm" {
		return ErrForbidden
	}
	return s.repo.DeleteShopItem(ctx, id)
}
