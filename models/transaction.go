package models

import (
	"time"

	"github.com/google/uuid"
)

type Transaction struct {
	ID            uuid.UUID        `json:"id"`
	CampaignID    uuid.UUID        `json:"campaign_id"`
	CharacterID   uuid.UUID        `json:"character_id"`
	CharacterName string           `json:"character_name"`
	Type          string           `json:"type"`
	Status        string           `json:"status"`
	OriginalTotal float64          `json:"original_total"`
	AdjustedTotal float64          `json:"adjusted_total"`
	TotalCoinID   uuid.UUID        `json:"total_coin_id"`
	TotalCoin     string           `json:"total_coin"`
	Notes         string           `json:"notes"`
	CreatedBy     uuid.UUID        `json:"created_by"`
	CreatedAt     time.Time        `json:"created_at"`
	ConfirmedAt   *time.Time       `json:"confirmed_at,omitempty"`
	Items         []TransactionItem `json:"items"`
}

type TransactionItem struct {
	ID                uuid.UUID  `json:"id"`
	TransactionID     uuid.UUID  `json:"transaction_id"`
	ShopItemID        *uuid.UUID `json:"shop_item_id,omitempty"`
	InventoryItemID   *uuid.UUID `json:"inventory_item_id,omitempty"`
	Name              string     `json:"name"`
	Quantity          int        `json:"quantity"`
	UnitValue         float64    `json:"unit_value"`
	AdjustedUnitValue float64    `json:"adjusted_unit_value"`
	LineTotal         float64    `json:"line_total"`
	CoinID            uuid.UUID  `json:"coin_id"`
	CoinAbbreviation  string     `json:"coin_abbreviation"`
}
