package asset

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/dungpd/seta/core-service/internal/cache"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog/log"
)

type Service interface {
	Create(ctx context.Context, callerID string, parentID *string, assetType, title string, content *string) (*Asset, error)
	GetByID(ctx context.Context, callerID, assetID string) (*Asset, error)
	Update(ctx context.Context, callerID, assetID, title string, content *string) (*Asset, error)
	Delete(ctx context.Context, callerID, assetID string) error
	Share(ctx context.Context, callerID, assetID, targetUserID, accessLevel string) error
	RevokeShare(ctx context.Context, callerID, assetID, targetUserID string) error
	List(ctx context.Context, callerID string, page, limit int) ([]*Asset, int64, error)
}

type service struct {
	repo      Repository
	publisher Publisher
	rdb       *redis.Client
}

func NewService(repo Repository, rdb *redis.Client, publisher Publisher) Service {
	return &service{repo: repo, publisher: publisher, rdb: rdb}
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
	assetCacheKey := cache.AssetKey(assetID)
	cached, err := s.rdb.Get(ctx, assetCacheKey).Result()
	var existing *Asset
	if err == nil {
		if jsonErr := json.Unmarshal([]byte(cached), &existing); jsonErr != nil {
			log.Warn().Err(jsonErr).Str("asset_id", assetID).Msg("failed to unmarshal cached asset")
			existing = nil
		}
	}

	if existing == nil {
		existing, err = s.repo.GetByID(ctx, assetID)
		if err != nil {
			return nil, err
		}
		if data, marshalErr := json.Marshal(existing); marshalErr != nil {
			log.Warn().Err(marshalErr).Str("asset_id", assetID).Msg("failed to marshal asset for cache")
		} else {
			s.rdb.Set(ctx, assetCacheKey, data, 5*time.Minute)
		}
	}

	if existing.OwnerID == callerID {
		return existing, nil
	}

	aclEntries, err := s.cachedACL(ctx, assetID)
	if err != nil {
		return nil, err
	}
	for _, a := range aclEntries {
		if a.UserID == callerID {
			return existing, nil
		}
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

func (s *service) cachedACL(ctx context.Context, assetID string) ([]*AssetACL, error) {
	cacheKey := cache.AssetACLKey(assetID)
	if cached, err := s.rdb.Get(ctx, cacheKey).Result(); err == nil {
		var entries []*AssetACL
		if jsonErr := json.Unmarshal([]byte(cached), &entries); jsonErr != nil {
			log.Warn().Err(jsonErr).Str("asset_id", assetID).Msg("failed to unmarshal cached ACL")
		} else {
			return entries, nil
		}
	}
	entries, err := s.repo.ListACLByAsset(ctx, assetID)
	if err != nil {
		return nil, err
	}
	if data, marshalErr := json.Marshal(entries); marshalErr != nil {
		log.Warn().Err(marshalErr).Str("asset_id", assetID).Msg("failed to marshal ACL for cache")
	} else {
		s.rdb.Set(ctx, cacheKey, data, 5*time.Minute)
	}
	return entries, nil
}

func (s *service) Update(ctx context.Context, callerID, assetID, title string, content *string) (*Asset, error) {
	existing, err := s.repo.GetByID(ctx, assetID)
	if err != nil {
		return nil, err
	}
	if existing.Type == AssetTypeFolder && content != nil {
		return nil, ErrFolderContentNotAllowed
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
	s.rdb.Del(ctx, cache.AssetKey(assetID))
	return updated, nil
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

func (s *service) Delete(ctx context.Context, callerID, assetID string) error {
	existing, err := s.repo.GetByID(ctx, assetID)
	if err != nil {
		return err
	}
	if existing.OwnerID != callerID {
		return ErrForbidden
	}
	var descendants []string
	if existing.Type == AssetTypeFolder {
		var descErr error
		descendants, descErr = s.repo.GetDescendantIDs(ctx, assetID)
		if descErr != nil {
			log.Warn().Err(descErr).Str("asset_id", assetID).Msg("failed to fetch descendants for cache invalidation")
		}
	}
	if err := s.repo.Delete(ctx, assetID); err != nil {
		return err
	}
	s.rdb.Del(ctx, cache.AssetKey(assetID))
	s.rdb.Del(ctx, cache.AssetACLKey(assetID))
	for _, id := range descendants {
		s.rdb.Del(ctx, cache.AssetKey(id))
		s.rdb.Del(ctx, cache.AssetACLKey(id))
	}
	if existing.Type == AssetTypeFolder {
		s.publishEvent(ctx, EventFolderDeleted, assetID, existing.OwnerID)
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
	descendants, err := s.repo.UpsertACLWithCascade(ctx, assetID, asset.Type, targetUserID, accessLevel)
	if err != nil {
		return err
	}
	if asset.Type == AssetTypeFolder {
		s.publishEvent(ctx, EventFolderShared, assetID, asset.OwnerID)
		for _, id := range descendants {
			s.rdb.Del(ctx, cache.AssetACLKey(id))
		}
	}
	s.rdb.Del(ctx, cache.AssetACLKey(assetID))
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
	descendants, err := s.repo.DeleteACLWithCascade(ctx, assetID, asset.Type, targetUserID)
	if err != nil {
		return err
	}
	for _, id := range descendants {
		s.rdb.Del(ctx, cache.AssetACLKey(id))
	}
	s.rdb.Del(ctx, cache.AssetACLKey(assetID))
	return nil
}

func (s *service) List(ctx context.Context, callerID string, page, limit int) ([]*Asset, int64, error) {
	total, err := s.repo.CountByOwner(ctx, callerID)
	if err != nil {
		return nil, 0, err
	}
	offset := int32((page - 1) * limit)
	assets, err := s.repo.List(ctx, callerID, int32(limit), offset)
	if err != nil {
		return nil, 0, err
	}
	return assets, total, nil
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
