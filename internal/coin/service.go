package coin

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

// ── CoinTypes ─────────────────────────────────────────────────────────────────

func (s *Service) CreateCoinType(ctx context.Context, campaignID uuid.UUID, name, abbreviation, emoji string) (*models.CoinType, error) {
	if name == "" {
		return nil, fmt.Errorf("name is required")
	}
	if abbreviation == "" {
		return nil, fmt.Errorf("abbreviation is required")
	}
	return s.repo.CreateCoinType(ctx, campaignID, name, abbreviation, emoji)
}

func (s *Service) GetCoinType(ctx context.Context, id uuid.UUID) (*models.CoinType, error) {
	return s.repo.GetCoinTypeByID(ctx, id)
}

func (s *Service) GetCoinByID(ctx context.Context, id uuid.UUID) (*models.CoinType, error) {
	return s.repo.GetCoinTypeByID(ctx, id)
}

func (s *Service) ListCoinTypes(ctx context.Context, campaignID uuid.UUID) ([]models.CoinType, error) {
	return s.repo.ListCoinTypes(ctx, campaignID)
}

func (s *Service) UpdateCoinType(ctx context.Context, id uuid.UUID, name, abbreviation, emoji string) (*models.CoinType, error) {
	if name == "" {
		return nil, fmt.Errorf("name is required")
	}
	if abbreviation == "" {
		return nil, fmt.Errorf("abbreviation is required")
	}
	return s.repo.UpdateCoinType(ctx, id, name, abbreviation, emoji)
}

func (s *Service) DeleteCoinType(ctx context.Context, id uuid.UUID) error {
	inUse, err := s.repo.IsInUse(ctx, id)
	if err != nil {
		return err
	}
	if inUse {
		return ErrCoinInUse
	}
	return s.repo.DeleteCoinType(ctx, id)
}

func (s *Service) SetDefaultCoin(ctx context.Context, campaignID, coinID uuid.UUID) error {
	return s.repo.SetDefaultCoin(ctx, campaignID, coinID)
}

func (s *Service) GetDefaultCoin(ctx context.Context, campaignID uuid.UUID) (*models.CoinType, error) {
	return s.repo.GetDefaultCoin(ctx, campaignID)
}

// ── CoinConversions ───────────────────────────────────────────────────────────

func (s *Service) CreateConversion(ctx context.Context, campaignID, fromID, toID uuid.UUID, rate float64) ([]models.CoinConversion, error) {
	if fromID == toID {
		return nil, ErrSameCoin
	}
	if rate <= 0 {
		return nil, fmt.Errorf("rate must be greater than zero")
	}
	exists, err := s.repo.PairExists(ctx, fromID, toID)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, ErrConversionExists
	}
	return s.repo.CreateConversionPair(ctx, campaignID, fromID, toID, rate)
}

func (s *Service) ListConversions(ctx context.Context, campaignID uuid.UUID) ([]models.CoinConversion, error) {
	return s.repo.ListConversions(ctx, campaignID)
}

func (s *Service) GetConversion(ctx context.Context, id uuid.UUID) (*models.CoinConversion, error) {
	return s.repo.GetConversionByID(ctx, id)
}

func (s *Service) DeleteConversion(ctx context.Context, id uuid.UUID) error {
	return s.repo.DeleteConversionPair(ctx, id)
}
