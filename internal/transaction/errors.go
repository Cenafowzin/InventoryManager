package transaction

import "errors"

var (
	ErrTransactionNotFound  = errors.New("transaction not found")
	ErrNotDraft             = errors.New("transaction is not in draft status")
	ErrAlreadyConfirmed     = errors.New("transaction is already confirmed or cancelled")
	ErrForbidden            = errors.New("forbidden")
	ErrInsufficientFunds    = errors.New("insufficient funds")
	ErrItemUnavailable      = errors.New("shop item is not available for purchase")
	ErrInventoryItemMissing = errors.New("inventory item not found or insufficient quantity")
	ErrConflictingAdjust    = errors.New("cannot adjust total and individual items in the same request")
	ErrNoItems              = errors.New("transaction must have at least one item")
	ErrNoConversion         = errors.New("no conversion path found between coins")
)
