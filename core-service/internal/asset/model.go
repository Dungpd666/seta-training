package asset

import (
	"errors"
	"time"
)

var (
	ErrNotFound    = errors.New("asset not found")
	ErrForbidden   = errors.New("forbidden: insufficient permissions")
	ErrInvalidType = errors.New("invalid asset type")
)

type AssetACL struct {
	AssetID     string `json:"asset_id"`
	UserID      string `json:"user_id"`
	AccessLevel string `json:"access_level"`
}

type Asset struct {
	AssetID   string    `json:"asset_id"`
	OwnerID   string    `json:"owner_id"`
	ParentID  *string   `json:"parent_id,omitempty"`
	Type      string    `json:"type"`
	Title     string    `json:"title"`
	Content   *string   `json:"content,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}
