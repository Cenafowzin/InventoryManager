package shop

import (
	"context"

	"github.com/google/uuid"

	"github.com/rubendubeux/inventory-manager/models"
)

type CategoryService interface {
	ValidateCategoryIDs(ctx context.Context, campaignID uuid.UUID, categoryIDs []uuid.UUID) error
	SetShopItemCategories(ctx context.Context, shopItemID uuid.UUID, categoryIDs []uuid.UUID) error
}

type CoinService interface {
	GetDefaultCoin(ctx context.Context, campaignID uuid.UUID) (*models.CoinType, error)
	GetCoinByID(ctx context.Context, id uuid.UUID) (*models.CoinType, error)
}

type Service struct {
	repo        *Repository
	categorySvc CategoryService
	coinSvc     CoinService
}

func NewService(repo *Repository, categorySvc CategoryService, coinSvc CoinService) *Service {
	return &Service{repo: repo, categorySvc: categorySvc, coinSvc: coinSvc}
}

// ── Shops ─────────────────────────────────────────────────────────────────────

func (s *Service) ListShops(ctx context.Context, campaignID uuid.UUID) ([]models.Shop, error) {
	return s.repo.ListShops(ctx, campaignID)
}

func (s *Service) CreateShop(ctx context.Context, campaignID uuid.UUID, requesterRole, name, color string) (*models.Shop, error) {
	if requesterRole != "gm" {
		return nil, ErrForbidden
	}
	return s.repo.CreateShop(ctx, campaignID, name, color)
}

func (s *Service) UpdateShop(ctx context.Context, id uuid.UUID, requesterRole, name, color string, isActive bool) (*models.Shop, error) {
	if requesterRole != "gm" {
		return nil, ErrForbidden
	}
	return s.repo.UpdateShop(ctx, id, name, color, isActive)
}

func (s *Service) DeleteShop(ctx context.Context, id uuid.UUID, requesterRole string) error {
	if requesterRole != "gm" {
		return ErrForbidden
	}
	return s.repo.DeleteShop(ctx, id)
}

// ── Shop items ────────────────────────────────────────────────────────────────

type CreateShopItemInput struct {
	Name          string
	Description   string
	Emoji         string
	WeightKg      float64
	BaseValue     float64
	ValueCoinID   *uuid.UUID
	ShopID        *uuid.UUID
	StockQuantity *int
	IsAvailable   bool
	CategoryIDs   []uuid.UUID
}

type UpdateShopItemInput struct {
	Name          string
	Description   string
	Emoji         string
	WeightKg      float64
	BaseValue     float64
	ValueCoinID   *uuid.UUID
	ShopID        *uuid.UUID
	StockQuantity *int
	IsAvailable   bool
	CategoryIDs   []uuid.UUID
}

type ListShopItemsFilters struct {
	CategoryID *uuid.UUID
	ShopID     *uuid.UUID
}

func (s *Service) CreateShopItem(ctx context.Context, campaignID uuid.UUID, requesterRole string, input CreateShopItemInput) (*models.ShopItem, error) {
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

	item, err := s.repo.CreateShopItem(ctx, campaignID, valueCoinID, input.ShopID,
		input.Name, input.Description, input.Emoji,
		input.WeightKg, input.BaseValue, input.StockQuantity, input.IsAvailable)
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

func (s *Service) ListShopItems(ctx context.Context, campaignID uuid.UUID, requesterRole string, filters ListShopItemsFilters, includeUnavailable bool) ([]models.ShopItem, error) {
	onlyAvailable := requesterRole != "gm" || !includeUnavailable
	return s.repo.ListShopItems(ctx, campaignID, ShopFilters{
		OnlyAvailable: onlyAvailable,
		CategoryID:    filters.CategoryID,
		ShopID:        filters.ShopID,
	})
}

func (s *Service) GetShopItemByID(ctx context.Context, id uuid.UUID, requesterRole string) (*models.ShopItem, error) {
	item, err := s.repo.GetShopItemByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if requesterRole != "gm" && !item.IsAvailable {
		return nil, ErrShopItemNotFound
	}
	return item, nil
}

func (s *Service) UpdateShopItem(ctx context.Context, id, campaignID uuid.UUID, requesterRole string, input UpdateShopItemInput) (*models.ShopItem, error) {
	if requesterRole != "gm" {
		return nil, ErrForbidden
	}

	if input.ValueCoinID != nil {
		coin, err := s.coinSvc.GetCoinByID(ctx, *input.ValueCoinID)
		if err != nil {
			return nil, err
		}
		if coin.CampaignID != campaignID {
			return nil, ErrCoinNotInCampaign
		}
	}

	item, err := s.repo.UpdateShopItem(ctx, id, input.ValueCoinID, input.ShopID,
		input.Name, input.Description, input.Emoji,
		input.WeightKg, input.BaseValue, input.StockQuantity, input.IsAvailable)
	if err != nil {
		return nil, err
	}

	if input.CategoryIDs != nil {
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

func (s *Service) DeleteShopItem(ctx context.Context, id uuid.UUID, requesterRole string) error {
	if requesterRole != "gm" {
		return ErrForbidden
	}
	return s.repo.DeleteShopItem(ctx, id)
}
