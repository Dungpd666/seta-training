package asset

import (
	"errors"
	"time"
)

var (
	ErrNotFound    = errors.New("asset not found")
	ErrForbidden   = errors.New("only owner can modify this asset")
	ErrInvalidType = errors.New("invalid asset type")
)

type Asset struct {
	AssetID   string    `json:"asset_id"`
	OwnerID   string    `json:"owner_id"`
	ParentID  *string   `json:"parent_id,omitempty"`
	Type      string    `json:"type"`
	Title     string    `json:"title"`
	Content   *string   `json:"content,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}
