package character

import "errors"

var (
	ErrCharacterNotFound = errors.New("character not found")
	ErrForbidden         = errors.New("forbidden")
	ErrOwnerNotMember    = errors.New("owner_user_id is not a member of this campaign")
)
