package asset

import (
	"context"
	"errors"
	"time"
)

const (
	AssetTypeFolder  = "folder"
	AssetTypeNote    = "note"
	AccessLevelRead  = "read"
	AccessLevelWrite = "write"
)

var (
	ErrNotFound                = errors.New("asset not found")
	ErrForbidden               = errors.New("forbidden: insufficient permissions")
	ErrInvalidType             = errors.New("invalid asset type")
	ErrTargetUserNotFound      = errors.New("target user not found")
	ErrParentNotFound          = errors.New("parent asset not found")
	ErrParentNotFolder         = errors.New("parent asset is not a folder")
	ErrNoteRequiresParent      = errors.New("note asset requires a parent folder")
	ErrFolderContentNotAllowed = errors.New("folder asset cannot have content")
	ErrMaxDepthExceeded        = errors.New("maximum folder depth exceeded")
)

const TopicAssetChanges = "asset.changes"

const (
	EventNoteCreated   = "NOTE_CREATED"
	EventNoteUpdated   = "NOTE_UPDATED"
	EventFolderDeleted = "FOLDER_DELETED"
	EventFolderShared  = "FOLDER_SHARED"
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

type AssetEvent struct {
	Event     string    `json:"event"`
	AssetID   string    `json:"asset_id"`
	OwnerID   string    `json:"owner_id"`
	Timestamp time.Time `json:"timestamp"`
}

type CreateAssetRequest struct {
	ParentID *string `json:"parent_id"`
	Type     string  `json:"type"  binding:"required,oneof=folder note"`
	Title    string  `json:"title" binding:"required"`
	Content  *string `json:"content"`
}

type UpdateAssetRequest struct {
	Title   string  `json:"title"   binding:"required"`
	Content *string `json:"content"`
}

type ShareAssetRequest struct {
	UserID      string `json:"user_id" binding:"required"`
	AccessLevel string `json:"access"  binding:"required,oneof=read write"`
}

type Publisher interface {
	Publish(ctx context.Context, topic string, payload any) error
}
