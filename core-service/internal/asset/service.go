package asset

import (
	"context"
	"errors"
	"time"

	"github.com/rs/zerolog/log"
)

type Service interface {
	Create(ctx context.Context, callerID string, parentID *string, assetType, title string, content *string) (*Asset, error)
	GetByID(ctx context.Context, callerID, assetID string) (*Asset, error)
	Update(ctx context.Context, callerID, assetID, title string, content *string) (*Asset, error)
	Delete(ctx context.Context, callerID, assetID string) error
	Share(ctx context.Context, callerID, assetID, targetUserID, accessLevel string) error
	RevokeShare(ctx context.Context, callerID, assetID, targetUserID string) error
}

type service struct {
	repo      Repository
	publisher Publisher
}

func NewService(repo Repository, publisher Publisher) Service {
	return &service{repo: repo, publisher: publisher}
}

func (s *service) Create(ctx context.Context, callerID string, parentID *string, assetType, title string, content *string) (*Asset, error) {
	if assetType != AssetTypeNote && assetType != AssetTypeFolder {
		return nil, ErrInvalidType
	}
	if assetType == AssetTypeNote && parentID == nil {
		return nil, ErrNoteRequiresParent
	}
	if assetType == AssetTypeFolder && content != nil {
		return nil, ErrFolderContentNotAllowed
	}
	if parentID != nil {
		parent, err := s.repo.GetByID(ctx, *parentID)
		if errors.Is(err, ErrNotFound) {
			return nil, ErrParentNotFound
		}
		if err != nil {
			return nil, err
		}
		if parent.Type != AssetTypeFolder {
			return nil, ErrParentNotFolder
		}
		if err := s.requireWriteAccess(ctx, parent, callerID); err != nil {
			return nil, err
		}
	}
	created, err := s.repo.Create(ctx, callerID, parentID, assetType, title, content)
	if err != nil {
		return nil, err
	}
	if assetType == AssetTypeNote {
		s.publishEvent(ctx, EventNoteCreated, created.AssetID, created.OwnerID)
	}
	return created, nil
}

func (s *service) GetByID(ctx context.Context, callerID, assetID string) (*Asset, error) {
	existing, err := s.repo.GetByID(ctx, assetID)
	if err != nil {
		return nil, err
	}
	if existing.OwnerID == callerID {
		return existing, nil
	}
	acl, err := s.repo.GetACLEntry(ctx, assetID, callerID)
	if err != nil {
		return nil, err
	}
	if acl != nil {
		return existing, nil
	}
	isManager, err := s.repo.IsManagerOfOwner(ctx, callerID, existing.OwnerID)
	if err != nil {
		return nil, err
	}
	if isManager {
		return existing, nil
	}
	return nil, ErrForbidden
}

func (s *service) Update(ctx context.Context, callerID, assetID, title string, content *string) (*Asset, error) {
	existing, err := s.repo.GetByID(ctx, assetID)
	if err != nil {
		return nil, err
	}
	if err := s.requireWriteAccess(ctx, existing, callerID); err != nil {
		return nil, err
	}
	updated, err := s.repo.Update(ctx, assetID, title, content)
	if err != nil {
		return nil, err
	}
	if existing.Type == AssetTypeNote {
		s.publishEvent(ctx, EventNoteUpdated, assetID, existing.OwnerID)
	}
	return updated, nil
}

func (s *service) Delete(ctx context.Context, callerID, assetID string) error {
	existing, err := s.repo.GetByID(ctx, assetID)
	if err != nil {
		return err
	}
	if existing.OwnerID != callerID {
		return ErrForbidden
	}
	if err := s.repo.Delete(ctx, assetID); err != nil {
		return err
	}
	if existing.Type == AssetTypeFolder {
		s.publishEvent(ctx, EventFolderDeleted, assetID, existing.OwnerID)
	}
	return nil
}

func (s *service) requireWriteAccess(ctx context.Context, asset *Asset, callerID string) error {
	if asset.OwnerID == callerID {
		return nil
	}
	acl, err := s.repo.GetACLEntry(ctx, asset.AssetID, callerID)
	if err != nil {
		return err
	}
	if acl == nil || acl.AccessLevel != AccessLevelWrite {
		return ErrForbidden
	}
	return nil
}

func (s *service) Share(ctx context.Context, callerID, assetID, targetUserID, accessLevel string) error {
	asset, err := s.repo.GetByID(ctx, assetID)
	if err != nil {
		return err
	}
	if asset.OwnerID != callerID {
		return ErrForbidden
	}
	exists, err := s.repo.UserExists(ctx, targetUserID)
	if err != nil {
		return err
	}
	if !exists {
		return ErrTargetUserNotFound
	}
	if err := s.applyACLCascade(ctx, assetID, asset.Type, func(id string) error {
		return s.repo.UpsertACLEntry(ctx, id, targetUserID, accessLevel)
	}); err != nil {
		return err
	}
	if asset.Type == AssetTypeFolder {
		s.publishEvent(ctx, EventFolderShared, assetID, asset.OwnerID)
	}
	return nil
}

func (s *service) RevokeShare(ctx context.Context, callerID, assetID, targetUserID string) error {
	asset, err := s.repo.GetByID(ctx, assetID)
	if err != nil {
		return err
	}
	if asset.OwnerID != callerID {
		return ErrForbidden
	}
	return s.applyACLCascade(ctx, assetID, asset.Type, func(id string) error {
		return s.repo.DeleteACLEntry(ctx, id, targetUserID)
	})
}

func (s *service) publishEvent(ctx context.Context, eventType, assetID, ownerID string) {
	if err := s.publisher.Publish(ctx, TopicAssetChanges, AssetEvent{
		Event:     eventType,
		AssetID:   assetID,
		OwnerID:   ownerID,
		Timestamp: time.Now(),
	}); err != nil {
		log.Warn().Err(err).Str("event", eventType).Msg("failed to publish asset event")
	}
}

func (s *service) applyACLCascade(ctx context.Context, assetID, assetType string, apply func(string) error) error {
	if err := apply(assetID); err != nil {
		return err
	}
	if assetType != AssetTypeFolder {
		return nil
	}
	descendants, err := s.repo.GetDescendantIDs(ctx, assetID)
	if err != nil {
		return err
	}
	for _, id := range descendants {
		if err := apply(id); err != nil {
			return err
		}
	}
	return nil
}
