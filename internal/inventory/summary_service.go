package inventory

import (
	"context"

	"github.com/google/uuid"

	"github.com/rubendubeux/inventory-manager/internal/character"
	"github.com/rubendubeux/inventory-manager/models"
)

type LoadSpace struct {
	ID               uuid.UUID `json:"id"`
	Name             string    `json:"name"`
	CountsTowardLoad bool      `json:"counts_toward_load"`
	CapacityKg       *float64  `json:"capacity_kg"`
	CurrentWeightKg  float64   `json:"current_weight_kg"`
	IsOverCapacity   bool      `json:"is_over_capacity"`
}

type LoadSummary struct {
	MaxCarryWeightKg *float64    `json:"max_carry_weight_kg"`
	CurrentLoadKg    float64     `json:"current_load_kg"`
	LoadPercentage   *float64    `json:"load_percentage"`
	IsOverloaded     bool        `json:"is_overloaded"`
	Spaces           []LoadSpace `json:"spaces"`
}

type InventorySummary struct {
	LoadSummary
	Items      []models.Item      `json:"items"`
	CoinPurse  []models.CoinPurse `json:"coin_purse"`
	TotalValue float64            `json:"total_value"`
}

type SummaryService struct {
	storageRepo *StorageRepository
	itemRepo    *ItemRepository
	coinRepo    *CoinRepository
	charRepo    *character.Repository
}

func NewSummaryService(storageRepo *StorageRepository, itemRepo *ItemRepository, coinRepo *CoinRepository, charRepo *character.Repository) *SummaryService {
	return &SummaryService{
		storageRepo: storageRepo,
		itemRepo:    itemRepo,
		coinRepo:    coinRepo,
		charRepo:    charRepo,
	}
}

func (s *SummaryService) GetLoad(ctx context.Context, characterID, requesterID uuid.UUID, requesterRole string) (*LoadSummary, error) {
	ch, err := s.charRepo.GetCharacterByID(ctx, characterID)
	if err != nil {
		return nil, err
	}
	if err := checkCharAccess(ch, requesterID, requesterRole); err != nil {
		return nil, err
	}

	spaces, err := s.storageRepo.ListStorageSpaces(ctx, characterID)
	if err != nil {
		return nil, err
	}

	summary := &LoadSummary{
		MaxCarryWeightKg: ch.MaxCarryWeightKg,
		Spaces:           make([]LoadSpace, 0, len(spaces)),
	}

	for _, ss := range spaces {
		ls := LoadSpace{
			ID:               ss.ID,
			Name:             ss.Name,
			CountsTowardLoad: ss.CountsTowardLoad,
			CapacityKg:       ss.CapacityKg,
			CurrentWeightKg:  ss.CurrentWeightKg,
			IsOverCapacity:   ss.CapacityKg != nil && ss.CurrentWeightKg > *ss.CapacityKg,
		}
		if ss.CountsTowardLoad {
			summary.CurrentLoadKg += ss.CurrentWeightKg
		}
		summary.Spaces = append(summary.Spaces, ls)
	}

	if ch.MaxCarryWeightKg != nil {
		pct := (summary.CurrentLoadKg / *ch.MaxCarryWeightKg) * 100
		summary.LoadPercentage = &pct
		summary.IsOverloaded = summary.CurrentLoadKg > *ch.MaxCarryWeightKg
	}

	return summary, nil
}

func (s *SummaryService) GetInventorySummary(ctx context.Context, characterID, requesterID uuid.UUID, requesterRole string) (*InventorySummary, error) {
	load, err := s.GetLoad(ctx, characterID, requesterID, requesterRole)
	if err != nil {
		return nil, err
	}

	items, err := s.itemRepo.ListItemsByCharacter(ctx, characterID, ItemFilters{})
	if err != nil {
		return nil, err
	}

	purse, err := s.coinRepo.GetCoinPurse(ctx, characterID)
	if err != nil {
		return nil, err
	}

	var totalValue float64
	for _, item := range items {
		totalValue += item.Value * float64(item.Quantity)
	}

	return &InventorySummary{
		LoadSummary: *load,
		Items:       items,
		CoinPurse:   purse,
		TotalValue:  totalValue,
	}, nil
}
