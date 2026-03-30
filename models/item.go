package models

import (
	"time"

	"github.com/google/uuid"
)

type Item struct {
	ID             uuid.UUID   `json:"id"`
	CharacterID    uuid.UUID   `json:"character_id"`
	StorageSpaceID *uuid.UUID  `json:"storage_space_id"`
	StorageSpace   string      `json:"storage_space,omitempty"`
	Name           string      `json:"name"`
	Description    string      `json:"description"`
	Emoji          string      `json:"emoji,omitempty"`
	WeightKg       float64     `json:"weight_kg"`
	Value          float64     `json:"value"`
	ValueCoinID    *uuid.UUID  `json:"value_coin_id"`
	ValueCoin      string      `json:"value_coin,omitempty"`
	Quantity       int         `json:"quantity"`
	ShopItemID     *uuid.UUID  `json:"shop_item_id,omitempty"`
	Categories     []Category  `json:"categories"`
	CreatedAt      time.Time   `json:"created_at"`
	UpdatedAt      time.Time   `json:"updated_at"`
}
