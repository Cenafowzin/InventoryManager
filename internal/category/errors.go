package category

import "errors"

var (
	ErrCategoryNotFound = errors.New("category not found")
	ErrDuplicateName    = errors.New("category name already exists in this campaign")
)
