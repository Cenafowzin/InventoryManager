package transaction

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/rubendubeux/inventory-manager/models"
)

// ── External dependency interfaces ───────────────────────────────────────────

type CharacterRepo interface {
	GetCharacterByID(ctx context.Context, id uuid.UUID) (*models.Character, error)
}

type ShopRepo interface {
	GetShopItemByID(ctx context.Context, id uuid.UUID) (*models.ShopItem, error)
}

type ShopCategoryRepo interface {
	GetShopItemCategories(ctx context.Context, shopItemID uuid.UUID) ([]models.Category, error)
}

type ItemRepo interface {
	GetItemByID(ctx context.Context, id uuid.UUID) (*models.Item, error)
}

type CoinRepo interface {
	GetCoinBalance(ctx context.Context, characterID, coinTypeID uuid.UUID) (float64, error)
	ListConversionEdges(ctx context.Context, characterID uuid.UUID) ([]ConversionEdge, error)
}

type StorageRepo interface {
	GetDefaultStorageSpace(ctx context.Context, characterID uuid.UUID) (*models.StorageSpace, error)
}

type CoinTypeLookup interface {
	GetCoinType(ctx context.Context, id uuid.UUID) (*models.CoinType, error)
	GetDefaultCoin(ctx context.Context, campaignID uuid.UUID) (*models.CoinType, error)
}

// ConversionEdge mirrors inventory.ConversionEdge to avoid import cycle.
type ConversionEdge struct {
	FromID uuid.UUID
	ToID   uuid.UUID
	Rate   float64
}

// ── Service ───────────────────────────────────────────────────────────────────

type Service struct {
	repo        *Repository
	charRepo    CharacterRepo
	shopRepo    ShopRepo
	shopCatRepo ShopCategoryRepo
	itemRepo    ItemRepo
	coinRepo    CoinRepo
	storageRepo StorageRepo
	coinTypeSvc CoinTypeLookup
}

func NewService(
	repo *Repository,
	charRepo CharacterRepo,
	shopRepo ShopRepo,
	shopCatRepo ShopCategoryRepo,
	itemRepo ItemRepo,
	coinRepo CoinRepo,
	storageRepo StorageRepo,
	coinTypeSvc CoinTypeLookup,
) *Service {
	return &Service{
		repo:        repo,
		charRepo:    charRepo,
		shopRepo:    shopRepo,
		shopCatRepo: shopCatRepo,
		itemRepo:    itemRepo,
		coinRepo:    coinRepo,
		storageRepo: storageRepo,
		coinTypeSvc: coinTypeSvc,
	}
}

// ── Helpers ───────────────────────────────────────────────────────────────────

func findRate(edges []ConversionEdge, fromID, toID uuid.UUID) (float64, bool) {
	if fromID == toID {
		return 1.0, true
	}
	type state struct {
		id   uuid.UUID
		rate float64
	}
	type edge struct {
		to   uuid.UUID
		rate float64
	}
	graph := make(map[uuid.UUID][]edge)
	for _, e := range edges {
		graph[e.FromID] = append(graph[e.FromID], edge{e.ToID, e.Rate})
	}
	visited := make(map[uuid.UUID]bool)
	queue := []state{{fromID, 1.0}}
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		if visited[cur.id] {
			continue
		}
		visited[cur.id] = true
		if cur.id == toID {
			return cur.rate, true
		}
		for _, nb := range graph[cur.id] {
			if !visited[nb.to] {
				queue = append(queue, state{nb.to, cur.rate * nb.rate})
			}
		}
	}
	return 0, false
}

func (s *Service) checkAccess(ctx context.Context, characterID, requesterID uuid.UUID, requesterRole string) (*models.Character, error) {
	ch, err := s.charRepo.GetCharacterByID(ctx, characterID)
	if err != nil {
		return nil, err
	}
	if requesterRole != "gm" && ch.OwnerUserID != requesterID {
		return nil, ErrForbidden
	}
	return ch, nil
}

// ── CreateDraft ───────────────────────────────────────────────────────────────

type ItemInput struct {
	ShopItemID      *uuid.UUID
	InventoryItemID *uuid.UUID
	Quantity        int
}

type CreateInput struct {
	CharacterID uuid.UUID
	Type        string
	TotalCoinID *uuid.UUID
	Items       []ItemInput
}

func (s *Service) CreateDraft(
	ctx context.Context,
	campaignID, requesterID uuid.UUID,
	requesterRole string,
	input CreateInput,
) (*models.Transaction, error) {
	if len(input.Items) == 0 {
		return nil, ErrNoItems
	}
	if input.Type != "buy" && input.Type != "sell" {
		return nil, fmt.Errorf("type must be 'buy' or 'sell'")
	}

	ch, err := s.checkAccess(ctx, input.CharacterID, requesterID, requesterRole)
	if err != nil {
		return nil, err
	}

	// Resolve total coin
	var totalCoinID uuid.UUID
	if input.TotalCoinID != nil {
		totalCoinID = *input.TotalCoinID
	} else {
		def, err := s.coinTypeSvc.GetDefaultCoin(ctx, ch.CampaignID)
		if err != nil {
			return nil, fmt.Errorf("get default coin: %w", err)
		}
		totalCoinID = def.ID
	}

	edges, err := s.coinRepo.ListConversionEdges(ctx, input.CharacterID)
	if err != nil {
		return nil, err
	}

	var draftItems []DraftItemInput
	var originalTotal float64

	for _, inp := range input.Items {
		if inp.Quantity <= 0 {
			return nil, fmt.Errorf("quantity must be greater than zero")
		}

		var draftItem DraftItemInput
		draftItem.Quantity = inp.Quantity

		switch input.Type {
		case "buy":
			if inp.ShopItemID == nil {
				return nil, fmt.Errorf("shop_item_id required for buy transactions")
			}
			si, err := s.shopRepo.GetShopItemByID(ctx, *inp.ShopItemID)
			if err != nil {
				return nil, err
			}
			if !si.IsAvailable {
				return nil, ErrItemUnavailable
			}
			coinID := totalCoinID
			unitValue := si.BaseValue
			if si.ValueCoinID != nil && *si.ValueCoinID != totalCoinID {
				rate, ok := findRate(edges, *si.ValueCoinID, totalCoinID)
				if !ok {
					return nil, ErrNoConversion
				}
				unitValue = si.BaseValue * rate
				coinID = totalCoinID
			} else if si.ValueCoinID != nil {
				coinID = *si.ValueCoinID
			}
			draftItem.ShopItemID = inp.ShopItemID
			draftItem.Name = si.Name
			draftItem.UnitValue = unitValue
			draftItem.CoinID = coinID

		case "sell":
			if inp.InventoryItemID == nil {
				return nil, fmt.Errorf("inventory_item_id required for sell transactions")
			}
			item, err := s.itemRepo.GetItemByID(ctx, *inp.InventoryItemID)
			if err != nil {
				return nil, err
			}
			if item.CharacterID != input.CharacterID {
				return nil, ErrForbidden
			}
			coinID := totalCoinID
			unitValue := item.Value
			if item.ValueCoinID != nil && *item.ValueCoinID != totalCoinID {
				rate, ok := findRate(edges, *item.ValueCoinID, totalCoinID)
				if !ok {
					return nil, ErrNoConversion
				}
				unitValue = item.Value * rate
				coinID = totalCoinID
			} else if item.ValueCoinID != nil {
				coinID = *item.ValueCoinID
			}
			draftItem.InventoryItemID = inp.InventoryItemID
			draftItem.Name = item.Name
			draftItem.UnitValue = unitValue
			draftItem.CoinID = coinID
		}

		originalTotal += draftItem.UnitValue * float64(draftItem.Quantity)
		draftItems = append(draftItems, draftItem)
	}

	return s.repo.CreateDraft(ctx, campaignID, input.CharacterID, totalCoinID, requesterID, input.Type, originalTotal, draftItems)
}

// ── List / Get ────────────────────────────────────────────────────────────────

func (s *Service) List(ctx context.Context, campaignID, requesterID uuid.UUID, requesterRole string, f ListFilters) ([]models.Transaction, error) {
	if requesterRole != "gm" {
		f.RequesterID = &requesterID
	}
	txs, err := s.repo.List(ctx, campaignID, f)
	if err != nil {
		return nil, err
	}
	if txs == nil {
		return []models.Transaction{}, nil
	}
	return txs, nil
}

func (s *Service) Get(ctx context.Context, txID, requesterID uuid.UUID, requesterRole string) (*models.Transaction, error) {
	t, err := s.repo.GetByID(ctx, txID)
	if err != nil {
		return nil, err
	}
	if _, err := s.checkAccess(ctx, t.CharacterID, requesterID, requesterRole); err != nil {
		return nil, err
	}
	return t, nil
}

// ── Adjust ────────────────────────────────────────────────────────────────────

type ItemAdjustInput struct {
	ItemID            uuid.UUID
	AdjustedUnitValue float64
}

type AdjustInput struct {
	AdjustedTotal *float64
	Notes         *string
	Items         []ItemAdjustInput
}

func (s *Service) Adjust(ctx context.Context, txID, requesterID uuid.UUID, requesterRole string, patch AdjustInput) (*models.Transaction, error) {
	if patch.AdjustedTotal != nil && len(patch.Items) > 0 {
		return nil, ErrConflictingAdjust
	}

	t, err := s.repo.GetByID(ctx, txID)
	if err != nil {
		return nil, err
	}
	if t.Status != "draft" {
		return nil, ErrNotDraft
	}
	if _, err := s.checkAccess(ctx, t.CharacterID, requesterID, requesterRole); err != nil {
		return nil, err
	}

	if patch.AdjustedTotal != nil {
		return s.repo.AdjustTotal(ctx, txID, *patch.AdjustedTotal, patch.Notes)
	}

	if len(patch.Items) > 0 {
		adjustments := make([]ItemAdjustment, len(patch.Items))
		for i, it := range patch.Items {
			adjustments[i] = ItemAdjustment{ItemID: it.ItemID, AdjustedUnitValue: it.AdjustedUnitValue}
		}
		result, err := s.repo.AdjustItems(ctx, txID, adjustments)
		if err != nil {
			return nil, err
		}
		if patch.Notes != nil {
			if err := s.repo.UpdateNotes(ctx, txID, *patch.Notes); err != nil {
				return nil, err
			}
			result.Notes = *patch.Notes
		}
		return result, nil
	}

	// Only notes updated
	if patch.Notes != nil {
		if err := s.repo.UpdateNotes(ctx, txID, *patch.Notes); err != nil {
			return nil, err
		}
	}
	return s.repo.GetByID(ctx, txID)
}

// ── Confirm ───────────────────────────────────────────────────────────────────

func (s *Service) Confirm(ctx context.Context, txID, requesterID uuid.UUID, requesterRole string) (*models.Transaction, error) {
	t, err := s.repo.GetByID(ctx, txID)
	if err != nil {
		return nil, err
	}
	if t.Status != "draft" {
		return nil, ErrNotDraft
	}
	ch, err := s.checkAccess(ctx, t.CharacterID, requesterID, requesterRole)
	if err != nil {
		return nil, err
	}
	_ = ch

	switch t.Type {
	case "buy":
		return s.confirmBuy(ctx, t)
	case "sell":
		return s.confirmSell(ctx, t)
	default:
		return nil, fmt.Errorf("unknown transaction type: %s", t.Type)
	}
}

func (s *Service) confirmBuy(ctx context.Context, t *models.Transaction) (*models.Transaction, error) {
	// Check balance
	balance, err := s.coinRepo.GetCoinBalance(ctx, t.CharacterID, t.TotalCoinID)
	if err != nil {
		return nil, err
	}
	if balance < t.AdjustedTotal {
		return nil, ErrInsufficientFunds
	}

	// Get default storage
	storage, err := s.storageRepo.GetDefaultStorageSpace(ctx, t.CharacterID)
	if err != nil {
		return nil, fmt.Errorf("get default storage: %w", err)
	}

	// Build buy items with categories
	var buyItems []BuyItemData
	for _, ti := range t.Items {
		if ti.ShopItemID == nil {
			continue
		}
		si, err := s.shopRepo.GetShopItemByID(ctx, *ti.ShopItemID)
		if err != nil {
			return nil, err
		}
		cats, err := s.shopCatRepo.GetShopItemCategories(ctx, *ti.ShopItemID)
		if err != nil {
			return nil, err
		}
		catIDs := make([]uuid.UUID, len(cats))
		for i, c := range cats {
			catIDs[i] = c.ID
		}
		buyItems = append(buyItems, BuyItemData{
			Name:        si.Name,
			Description: si.Description,
			Emoji:       si.Emoji,
			WeightKg:    si.WeightKg,
			Value:       si.BaseValue,
			ValueCoinID: &t.TotalCoinID,
			Quantity:    ti.Quantity,
			ShopItemID:  *ti.ShopItemID,
			CategoryIDs: catIDs,
		})
	}

	return s.repo.ConfirmBuy(ctx, t.ID, t.CharacterID, storage.ID, t.TotalCoinID, t.AdjustedTotal, buyItems)
}

func (s *Service) confirmSell(ctx context.Context, t *models.Transaction) (*models.Transaction, error) {
	var sellItems []SellItemData
	for _, ti := range t.Items {
		if ti.InventoryItemID == nil {
			continue
		}
		sellItems = append(sellItems, SellItemData{
			InventoryItemID: *ti.InventoryItemID,
			Quantity:        ti.Quantity,
		})
	}

	return s.repo.ConfirmSell(ctx, t.ID, t.CharacterID, t.TotalCoinID, t.AdjustedTotal, sellItems)
}

// ── Cancel ────────────────────────────────────────────────────────────────────

func (s *Service) Cancel(ctx context.Context, txID, requesterID uuid.UUID, requesterRole string) (*models.Transaction, error) {
	t, err := s.repo.GetByID(ctx, txID)
	if err != nil {
		return nil, err
	}
	if t.Status != "draft" {
		return nil, ErrNotDraft
	}
	if _, err := s.checkAccess(ctx, t.CharacterID, requesterID, requesterRole); err != nil {
		return nil, err
	}
	return s.repo.Cancel(ctx, txID)
}
