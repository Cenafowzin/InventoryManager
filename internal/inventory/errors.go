package inventory

import "errors"

var (
	ErrStorageNotFound       = errors.New("storage space not found")
	ErrCannotDeleteDefault   = errors.New("cannot delete the default storage space")
	ErrDuplicateStorageName  = errors.New("storage space name already exists for this character")
	ErrItemNotFound          = errors.New("item not found")
	ErrStorageNotOwned       = errors.New("storage space does not belong to this character")
	ErrCategoryNotInCampaign = errors.New("one or more category_ids do not belong to this campaign")
	ErrInsufficientFunds     = errors.New("insufficient funds")
	ErrNoConversion          = errors.New("no conversion rate found between these coins")
	ErrForbidden             = errors.New("forbidden")
	ErrInsufficientQuantity  = errors.New("transfer quantity exceeds item quantity")
	ErrSameCharacter         = errors.New("source and target characters must be different")
)
