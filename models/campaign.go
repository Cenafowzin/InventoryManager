package models

import (
	"time"

	"github.com/google/uuid"
)

type Campaign struct {
	ID            uuid.UUID `json:"id"`
	Name          string    `json:"name"`
	Description   string    `json:"description"`
	CreatorUserID uuid.UUID `json:"creator_user_id"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

type CampaignInvite struct {
	ID         uuid.UUID  `json:"id"`
	CampaignID uuid.UUID  `json:"campaign_id"`
	Code       string     `json:"code"`
	CreatedBy  uuid.UUID  `json:"created_by"`
	ExpiresAt  *time.Time `json:"expires_at"`
	UsedAt     *time.Time `json:"used_at,omitempty"`
	UsedBy     *uuid.UUID `json:"used_by,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
}

type CampaignMember struct {
	ID         uuid.UUID `json:"id"`
	CampaignID uuid.UUID `json:"campaign_id"`
	UserID     uuid.UUID `json:"user_id"`
	Username   string    `json:"username"`
	Role       string    `json:"role"`
	JoinedAt   time.Time `json:"joined_at"`
}
