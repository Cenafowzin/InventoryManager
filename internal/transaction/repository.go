package transaction

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/rubendubeux/inventory-manager/models"
)

type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

// ── helpers ───────────────────────────────────────────────────────────────────

func (r *Repository) scanTransaction(row pgx.Row) (*models.Transaction, error) {
	var tx models.Transaction
	err := row.Scan(
		&tx.ID, &tx.CampaignID, &tx.CharacterID, &tx.CharacterName,
		&tx.Type, &tx.Status,
		&tx.OriginalTotal, &tx.AdjustedTotal,
		&tx.TotalCoinID, &tx.TotalCoin,
		&tx.Notes, &tx.CreatedBy, &tx.CreatedAt, &tx.ConfirmedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrTransactionNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("scan transaction: %w", err)
	}
	return &tx, nil
}

const txSelectSQL = `
	SELECT t.id, t.campaign_id, t.character_id, ch.name,
	       t.type, t.status,
	       t.original_total, t.adjusted_total,
	       t.total_coin_id, ct.abbreviation,
	       COALESCE(t.notes, ''), t.created_by, t.created_at, t.confirmed_at
	FROM transactions t
	JOIN characters ch ON ch.id = t.character_id
	JOIN coin_types  ct ON ct.id = t.total_coin_id
`

func (r *Repository) loadItems(ctx context.Context, txID uuid.UUID) ([]models.TransactionItem, error) {
	rows, err := r.db.Query(ctx, `
		SELECT ti.id, ti.transaction_id, ti.shop_item_id, ti.inventory_item_id,
		       ti.name, ti.quantity, ti.unit_value, ti.adjusted_unit_value,
		       ti.adjusted_unit_value * ti.quantity,
		       ti.coin_id, ct.abbreviation
		FROM transaction_items ti
		JOIN coin_types ct ON ct.id = ti.coin_id
		WHERE ti.transaction_id = $1
		ORDER BY ti.name
	`, txID)
	if err != nil {
		return nil, fmt.Errorf("load tx items: %w", err)
	}
	defer rows.Close()

	var items []models.TransactionItem
	for rows.Next() {
		var item models.TransactionItem
		if err := rows.Scan(
			&item.ID, &item.TransactionID, &item.ShopItemID, &item.InventoryItemID,
			&item.Name, &item.Quantity, &item.UnitValue, &item.AdjustedUnitValue,
			&item.LineTotal,
			&item.CoinID, &item.CoinAbbreviation,
		); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, nil
}

// ── CRUD ──────────────────────────────────────────────────────────────────────

type DraftItemInput struct {
	ShopItemID      *uuid.UUID
	InventoryItemID *uuid.UUID
	Name            string
	Quantity        int
	UnitValue       float64
	CoinID          uuid.UUID
}

func (r *Repository) CreateDraft(
	ctx context.Context,
	campaignID, characterID, totalCoinID, createdBy uuid.UUID,
	txType string,
	originalTotal float64,
	items []DraftItemInput,
) (*models.Transaction, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin: %w", err)
	}
	defer tx.Rollback(ctx)

	var txID uuid.UUID
	err = tx.QueryRow(ctx, `
		INSERT INTO transactions
		  (campaign_id, character_id, type, status, original_total, adjusted_total, total_coin_id, created_by)
		VALUES ($1, $2, $3, 'draft', $4, $4, $5, $6)
		RETURNING id
	`, campaignID, characterID, txType, originalTotal, totalCoinID, createdBy).Scan(&txID)
	if err != nil {
		return nil, fmt.Errorf("insert transaction: %w", err)
	}

	for _, item := range items {
		_, err = tx.Exec(ctx, `
			INSERT INTO transaction_items
			  (transaction_id, shop_item_id, inventory_item_id, name, quantity, unit_value, adjusted_unit_value, coin_id)
			VALUES ($1, $2, $3, $4, $5, $6, $6, $7)
		`, txID, item.ShopItemID, item.InventoryItemID, item.Name, item.Quantity, item.UnitValue, item.CoinID)
		if err != nil {
			return nil, fmt.Errorf("insert tx item: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit: %w", err)
	}

	return r.GetByID(ctx, txID)
}

func (r *Repository) GetByID(ctx context.Context, id uuid.UUID) (*models.Transaction, error) {
	t, err := r.scanTransaction(r.db.QueryRow(ctx, txSelectSQL+` WHERE t.id = $1`, id))
	if err != nil {
		return nil, err
	}
	t.Items, err = r.loadItems(ctx, t.ID)
	if err != nil {
		return nil, err
	}
	if t.Items == nil {
		t.Items = []models.TransactionItem{}
	}
	return t, nil
}

type ListFilters struct {
	CharacterID *uuid.UUID
	Status      *string
	Type        *string
	RequesterID *uuid.UUID // nil = gm (see all)
}

func (r *Repository) List(ctx context.Context, campaignID uuid.UUID, f ListFilters) ([]models.Transaction, error) {
	q := txSelectSQL + ` WHERE t.campaign_id = $1`
	args := []any{campaignID}
	i := 2

	if f.RequesterID != nil {
		q += fmt.Sprintf(` AND ch.owner_user_id = $%d`, i)
		args = append(args, *f.RequesterID)
		i++
	}
	if f.CharacterID != nil {
		q += fmt.Sprintf(` AND t.character_id = $%d`, i)
		args = append(args, *f.CharacterID)
		i++
	}
	if f.Status != nil {
		q += fmt.Sprintf(` AND t.status = $%d`, i)
		args = append(args, *f.Status)
		i++
	}
	if f.Type != nil {
		q += fmt.Sprintf(` AND t.type = $%d`, i)
		args = append(args, *f.Type)
		i++
	}
	_ = i

	q += ` ORDER BY t.created_at DESC`

	rows, err := r.db.Query(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("list transactions: %w", err)
	}
	defer rows.Close()

	var txs []models.Transaction
	for rows.Next() {
		t, err := r.scanTransaction(rows)
		if err != nil {
			return nil, err
		}
		t.Items = []models.TransactionItem{}
		txs = append(txs, *t)
	}
	return txs, nil
}

// ── Adjust ────────────────────────────────────────────────────────────────────

type ItemAdjustment struct {
	ItemID            uuid.UUID
	AdjustedUnitValue float64
}

// AdjustItems updates individual item values and recalculates adjusted_total.
func (r *Repository) AdjustItems(ctx context.Context, txID uuid.UUID, adjustments []ItemAdjustment) (*models.Transaction, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin: %w", err)
	}
	defer tx.Rollback(ctx)

	for _, a := range adjustments {
		tag, err := tx.Exec(ctx, `
			UPDATE transaction_items SET adjusted_unit_value = $1
			WHERE id = $2 AND transaction_id = $3
		`, a.AdjustedUnitValue, a.ItemID, txID)
		if err != nil {
			return nil, fmt.Errorf("update item: %w", err)
		}
		if tag.RowsAffected() == 0 {
			return nil, fmt.Errorf("transaction item %s not found", a.ItemID)
		}
	}

	// Recalculate adjusted_total
	_, err = tx.Exec(ctx, `
		UPDATE transactions
		SET adjusted_total = (
			SELECT COALESCE(SUM(adjusted_unit_value * quantity), 0)
			FROM transaction_items WHERE transaction_id = $1
		)
		WHERE id = $1
	`, txID)
	if err != nil {
		return nil, fmt.Errorf("recalculate total: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit: %w", err)
	}
	return r.GetByID(ctx, txID)
}

// AdjustTotal redistributes adjusted_total proportionally to all items.
func (r *Repository) AdjustTotal(ctx context.Context, txID uuid.UUID, newTotal float64, notes *string) (*models.Transaction, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin: %w", err)
	}
	defer tx.Rollback(ctx)

	// Get original_total to calculate factor
	var originalTotal float64
	if err := tx.QueryRow(ctx, `SELECT original_total FROM transactions WHERE id = $1`, txID).Scan(&originalTotal); err != nil {
		return nil, fmt.Errorf("get original total: %w", err)
	}

	if originalTotal > 0 {
		factor := newTotal / originalTotal
		_, err = tx.Exec(ctx, `
			UPDATE transaction_items
			SET adjusted_unit_value = GREATEST(0, unit_value * $1)
			WHERE transaction_id = $2
		`, factor, txID)
		if err != nil {
			return nil, fmt.Errorf("redistribute items: %w", err)
		}
	} else {
		// If original is 0, set all items to 0
		_, err = tx.Exec(ctx, `
			UPDATE transaction_items SET adjusted_unit_value = 0 WHERE transaction_id = $1
		`, txID)
		if err != nil {
			return nil, fmt.Errorf("zero items: %w", err)
		}
	}

	q := `UPDATE transactions SET adjusted_total = $1`
	args := []any{newTotal, txID}
	if notes != nil {
		q += `, notes = $3`
		args = append(args, *notes)
	}
	q += ` WHERE id = $2`
	if _, err := tx.Exec(ctx, q, args...); err != nil {
		return nil, fmt.Errorf("update total: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit: %w", err)
	}
	return r.GetByID(ctx, txID)
}

func (r *Repository) UpdateNotes(ctx context.Context, txID uuid.UUID, notes string) error {
	_, err := r.db.Exec(ctx, `UPDATE transactions SET notes = $1 WHERE id = $2`, notes, txID)
	return err
}

// ── Confirm / Cancel ──────────────────────────────────────────────────────────

type BuyItemData struct {
	Name        string
	Description string
	Emoji       string
	WeightKg    float64
	Value       float64
	ValueCoinID *uuid.UUID
	Quantity    int
	ShopItemID  uuid.UUID
	CategoryIDs []uuid.UUID
}

func (r *Repository) ConfirmBuy(
	ctx context.Context,
	txID, characterID, storageSpaceID, coinID uuid.UUID,
	totalToDebit float64,
	items []BuyItemData,
) (*models.Transaction, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin: %w", err)
	}
	defer tx.Rollback(ctx)

	for _, item := range items {
		var itemID uuid.UUID
		err = tx.QueryRow(ctx, `
			INSERT INTO items
			  (character_id, storage_space_id, shop_item_id, name, description, emoji,
			   weight_kg, value, value_coin_id, quantity)
			VALUES ($1, $2, $3, $4, NULLIF($5,''), NULLIF($6,''), $7, $8, $9, $10)
			RETURNING id
		`, characterID, storageSpaceID, item.ShopItemID,
			item.Name, item.Description, item.Emoji,
			item.WeightKg, item.Value, item.ValueCoinID, item.Quantity,
		).Scan(&itemID)
		if err != nil {
			return nil, fmt.Errorf("insert item: %w", err)
		}

		for _, catID := range item.CategoryIDs {
			_, err = tx.Exec(ctx, `
				INSERT INTO item_categories (item_id, category_id) VALUES ($1, $2) ON CONFLICT DO NOTHING
			`, itemID, catID)
			if err != nil {
				return nil, fmt.Errorf("insert item category: %w", err)
			}
		}
	}

	// Decrement stock_quantity for each shop item purchased
	for _, item := range items {
		_, err = tx.Exec(ctx, `
			UPDATE shop_items
			SET stock_quantity = GREATEST(0, stock_quantity - $1::int),
			    is_available   = CASE WHEN stock_quantity <= $1::int THEN false ELSE is_available END,
			    updated_at     = NOW()
			WHERE id = $2 AND stock_quantity IS NOT NULL
		`, item.Quantity, item.ShopItemID)
		if err != nil {
			return nil, fmt.Errorf("decrement stock: %w", err)
		}
	}

	// Debit coins
	_, err = tx.Exec(ctx, `
		INSERT INTO coin_purse (character_id, coin_type_id, amount, updated_at)
		VALUES ($1, $2, GREATEST(0, 0 - $3::numeric), NOW())
		ON CONFLICT (character_id, coin_type_id)
		DO UPDATE SET amount = GREATEST(0, coin_purse.amount - $3::numeric), updated_at = NOW()
	`, characterID, coinID, totalToDebit)
	if err != nil {
		return nil, fmt.Errorf("debit coins: %w", err)
	}

	now := time.Now()
	_, err = tx.Exec(ctx, `
		UPDATE transactions SET status = 'confirmed', confirmed_at = $1 WHERE id = $2
	`, now, txID)
	if err != nil {
		return nil, fmt.Errorf("confirm tx: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit: %w", err)
	}
	return r.GetByID(ctx, txID)
}

type SellItemData struct {
	InventoryItemID uuid.UUID
	Quantity        int
}

func (r *Repository) ConfirmSell(
	ctx context.Context,
	txID, characterID, coinID uuid.UUID,
	totalToCredit float64,
	items []SellItemData,
) (*models.Transaction, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin: %w", err)
	}
	defer tx.Rollback(ctx)

	for _, item := range items {
		var currentQty int
		err = tx.QueryRow(ctx, `SELECT quantity FROM items WHERE id = $1 AND character_id = $2`, item.InventoryItemID, characterID).Scan(&currentQty)
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrInventoryItemMissing
		}
		if err != nil {
			return nil, fmt.Errorf("get item qty: %w", err)
		}
		if currentQty < item.Quantity {
			return nil, ErrInventoryItemMissing
		}

		if currentQty == item.Quantity {
			_, err = tx.Exec(ctx, `DELETE FROM items WHERE id = $1`, item.InventoryItemID)
		} else {
			_, err = tx.Exec(ctx, `UPDATE items SET quantity = quantity - $1, updated_at = NOW() WHERE id = $2`, item.Quantity, item.InventoryItemID)
		}
		if err != nil {
			return nil, fmt.Errorf("update item qty: %w", err)
		}
	}

	// Credit coins
	_, err = tx.Exec(ctx, `
		INSERT INTO coin_purse (character_id, coin_type_id, amount, updated_at)
		VALUES ($1, $2, $3::numeric, NOW())
		ON CONFLICT (character_id, coin_type_id)
		DO UPDATE SET amount = coin_purse.amount + $3::numeric, updated_at = NOW()
	`, characterID, coinID, totalToCredit)
	if err != nil {
		return nil, fmt.Errorf("credit coins: %w", err)
	}

	now := time.Now()
	_, err = tx.Exec(ctx, `
		UPDATE transactions SET status = 'confirmed', confirmed_at = $1 WHERE id = $2
	`, now, txID)
	if err != nil {
		return nil, fmt.Errorf("confirm tx: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit: %w", err)
	}
	return r.GetByID(ctx, txID)
}

func (r *Repository) Cancel(ctx context.Context, txID uuid.UUID) (*models.Transaction, error) {
	tag, err := r.db.Exec(ctx, `
		UPDATE transactions SET status = 'cancelled' WHERE id = $1 AND status = 'draft'
	`, txID)
	if err != nil {
		return nil, fmt.Errorf("cancel: %w", err)
	}
	if tag.RowsAffected() == 0 {
		// either not found or not draft
		t, err2 := r.GetByID(ctx, txID)
		if err2 != nil {
			return nil, err2
		}
		if t.Status != "draft" {
			return nil, ErrNotDraft
		}
		return nil, ErrTransactionNotFound
	}
	return r.GetByID(ctx, txID)
}
