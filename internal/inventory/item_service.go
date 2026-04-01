package inventory

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/rubendubeux/inventory-manager/internal/category"
	"github.com/rubendubeux/inventory-manager/internal/character"
	"github.com/rubendubeux/inventory-manager/internal/coin"
	"github.com/rubendubeux/inventory-manager/models"
	"github.com/rubendubeux/inventory-manager/pkg/weight"
)

type ItemInput struct {
	Name           string
	Description    string
	Emoji          string
	WeightKg       float64
	WeightUnit     string
	Value          float64
	ValueCoinID    *uuid.UUID
	StorageSpaceID *uuid.UUID
	CategoryIDs    []uuid.UUID
	Quantity       int
	ShopItemID     *uuid.UUID
}

type ItemService struct {
	itemRepo    *ItemRepository
	storageRepo *StorageRepository
	charRepo    *character.Repository
	coinRepo    *coin.Repository
	catRepo     *category.Repository
}

func NewItemService(itemRepo *ItemRepository, storageRepo *StorageRepository, charRepo *character.Repository, coinRepo *coin.Repository, catRepo *category.Repository) *ItemService {
	return &ItemService{
		itemRepo:    itemRepo,
		storageRepo: storageRepo,
		charRepo:    charRepo,
		coinRepo:    coinRepo,
		catRepo:     catRepo,
	}
}

func (s *ItemService) CreateItem(ctx context.Context, characterID, requesterID uuid.UUID, requesterRole string, input ItemInput) (*models.Item, error) {
	ch, err := s.charRepo.GetCharacterByID(ctx, characterID)
	if err != nil {
		return nil, err
	}
	if err := checkCharAccess(ch, requesterID, requesterRole); err != nil {
		return nil, err
	}

	if input.Name == "" {
		return nil, fmt.Errorf("name is required")
	}
	if input.Quantity <= 0 {
		input.Quantity = 1
	}

	// Converte peso se necessário
	weightKg := weight.ToKg(input.WeightKg, input.WeightUnit)

	// Resolve storage space
	storageID, err := s.resolveStorage(ctx, characterID, input.StorageSpaceID)
	if err != nil {
		return nil, err
	}

	// Resolve moeda
	coinID, err := s.resolveCoin(ctx, ch.CampaignID, input.ValueCoinID)
	if err != nil {
		return nil, err
	}

	// Valida categorias
	if err := s.catRepo.ValidateCategoryIDs(ctx, ch.CampaignID, input.CategoryIDs); err != nil {
		return nil, err
	}

	item, err := s.itemRepo.CreateItem(ctx, characterID, storageID, input.ShopItemID, coinID, input.Name, input.Description, input.Emoji, weightKg, input.Value, input.Quantity)
	if err != nil {
		return nil, err
	}

	if len(input.CategoryIDs) > 0 {
		if err := s.catRepo.SetItemCategories(ctx, item.ID, input.CategoryIDs); err != nil {
			return nil, err
		}
	}

	item.Categories, _ = s.catRepo.GetItemCategories(ctx, item.ID)
	return item, nil
}

func (s *ItemService) GetItem(ctx context.Context, itemID, requesterID uuid.UUID, requesterRole string) (*models.Item, error) {
	item, err := s.itemRepo.GetItemByID(ctx, itemID)
	if err != nil {
		return nil, err
	}
	ch, err := s.charRepo.GetCharacterByID(ctx, item.CharacterID)
	if err != nil {
		return nil, err
	}
	if err := checkCharAccess(ch, requesterID, requesterRole); err != nil {
		return nil, err
	}
	item.Categories, _ = s.catRepo.GetItemCategories(ctx, item.ID)
	return item, nil
}

func (s *ItemService) ListItems(ctx context.Context, characterID, requesterID uuid.UUID, requesterRole string, filters ItemFilters) ([]models.Item, error) {
	ch, err := s.charRepo.GetCharacterByID(ctx, characterID)
	if err != nil {
		return nil, err
	}
	if err := checkCharAccess(ch, requesterID, requesterRole); err != nil {
		return nil, err
	}

	items, err := s.itemRepo.ListItemsByCharacter(ctx, characterID, filters)
	if err != nil {
		return nil, err
	}
	for i := range items {
		items[i].Categories, _ = s.catRepo.GetItemCategories(ctx, items[i].ID)
	}
	return items, nil
}

func (s *ItemService) UpdateItem(ctx context.Context, itemID, requesterID uuid.UUID, requesterRole string, input ItemInput) (*models.Item, error) {
	existing, err := s.itemRepo.GetItemByID(ctx, itemID)
	if err != nil {
		return nil, err
	}
	ch, err := s.charRepo.GetCharacterByID(ctx, existing.CharacterID)
	if err != nil {
		return nil, err
	}
	if err := checkCharAccess(ch, requesterID, requesterRole); err != nil {
		return nil, err
	}
	if input.Name == "" {
		return nil, fmt.Errorf("name is required")
	}
	if input.Quantity <= 0 {
		input.Quantity = 1
	}

	weightKg := weight.ToKg(input.WeightKg, input.WeightUnit)

	storageID, err := s.resolveStorage(ctx, existing.CharacterID, input.StorageSpaceID)
	if err != nil {
		return nil, err
	}

	coinID, err := s.resolveCoin(ctx, ch.CampaignID, input.ValueCoinID)
	if err != nil {
		return nil, err
	}

	if err := s.catRepo.ValidateCategoryIDs(ctx, ch.CampaignID, input.CategoryIDs); err != nil {
		return nil, err
	}

	item, err := s.itemRepo.UpdateItem(ctx, itemID, storageID, coinID, input.Name, input.Description, input.Emoji, weightKg, input.Value, input.Quantity)
	if err != nil {
		return nil, err
	}

	// Substitui categorias apenas se explicitamente fornecido
	if input.CategoryIDs != nil {
		if err := s.catRepo.SetItemCategories(ctx, item.ID, input.CategoryIDs); err != nil {
			return nil, err
		}
	}

	item.Categories, _ = s.catRepo.GetItemCategories(ctx, item.ID)
	return item, nil
}

func (s *ItemService) DeleteItem(ctx context.Context, itemID, requesterID uuid.UUID, requesterRole string) error {
	item, err := s.itemRepo.GetItemByID(ctx, itemID)
	if err != nil {
		return err
	}
	ch, err := s.charRepo.GetCharacterByID(ctx, item.CharacterID)
	if err != nil {
		return err
	}
	if err := checkCharAccess(ch, requesterID, requesterRole); err != nil {
		return err
	}

	// Se o item é um container (storage space linkado), reatribui itens ao default antes de deletar
	linkedSpace, err := s.storageRepo.GetStorageSpaceByItemID(ctx, itemID)
	if err != nil {
		return fmt.Errorf("check container: %w", err)
	}
	if linkedSpace != nil {
		defaultSpace, err := s.storageRepo.GetDefaultStorageSpace(ctx, item.CharacterID)
		if err != nil {
			return fmt.Errorf("get default space: %w", err)
		}
		if err := s.storageRepo.ReassignItemsToDefault(ctx, linkedSpace.ID, defaultSpace.ID); err != nil {
			return fmt.Errorf("reassign items: %w", err)
		}
		if err := s.storageRepo.DeleteStorageSpace(ctx, linkedSpace.ID); err != nil {
			return fmt.Errorf("delete linked space: %w", err)
		}
	}

	return s.itemRepo.DeleteItem(ctx, itemID)
}

// resolveStorage retorna o ID do espaço de armazenamento, usando o default se não informado.
func (s *ItemService) resolveStorage(ctx context.Context, characterID uuid.UUID, storageID *uuid.UUID) (uuid.UUID, error) {
	if storageID == nil {
		def, err := s.storageRepo.GetDefaultStorageSpace(ctx, characterID)
		if err != nil {
			return uuid.Nil, fmt.Errorf("get default storage: %w", err)
		}
		return def.ID, nil
	}
	// Valida que pertence ao personagem
	ss, err := s.storageRepo.GetStorageSpaceByID(ctx, *storageID)
	if err != nil {
		return uuid.Nil, err
	}
	if ss.CharacterID != characterID {
		return uuid.Nil, ErrStorageNotOwned
	}
	return *storageID, nil
}

// resolveCoin retorna o ID da moeda, usando a padrão da campanha se não informada.
func (s *ItemService) resolveCoin(ctx context.Context, campaignID uuid.UUID, coinID *uuid.UUID) (*uuid.UUID, error) {
	if coinID != nil {
		return coinID, nil
	}
	def, err := s.coinRepo.GetDefaultCoin(ctx, campaignID)
	if err != nil {
		// sem moeda padrão configurada — deixa nulo
		return nil, nil
	}
	return &def.ID, nil
}

func (s *ItemService) TransferItem(ctx context.Context, itemID, targetCharID, requesterID uuid.UUID, requesterRole string, quantity int) error {
	item, err := s.itemRepo.GetItemByID(ctx, itemID)
	if err != nil {
		return err
	}
	srcChar, err := s.charRepo.GetCharacterByID(ctx, item.CharacterID)
	if err != nil {
		return err
	}
	if err := checkCharAccess(srcChar, requesterID, requesterRole); err != nil {
		return err
	}
	tgtChar, err := s.charRepo.GetCharacterByID(ctx, targetCharID)
	if err != nil {
		return err
	}
	if tgtChar.CampaignID != srcChar.CampaignID {
		return ErrForbidden
	}
	if item.CharacterID == targetCharID {
		return ErrSameCharacter
	}
	if quantity <= 0 || quantity > item.Quantity {
		quantity = item.Quantity
	}
	return s.itemRepo.TransferItem(ctx, itemID, targetCharID, quantity)
}

func checkCharAccess(ch *models.Character, requesterID uuid.UUID, requesterRole string) error {
	if requesterRole == "gm" {
		return nil
	}
	if ch.OwnerUserID != requesterID {
		return ErrForbidden
	}
	return nil
}
