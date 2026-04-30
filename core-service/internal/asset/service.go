package asset

import "context"

type Service interface {
	Create(ctx context.Context, callerID string, parentID *string, assetType, title string, content *string) (*Asset, error)
	GetByID(ctx context.Context, callerID, assetID string) (*Asset, error)
	Update(ctx context.Context, callerID, assetID, title string, content *string) (*Asset, error)
	Delete(ctx context.Context, callerID, assetID string) error
	Share(ctx context.Context, callerID, assetID, targetUserID, accessLevel string) error
	RevokeShare(ctx context.Context, callerID, assetID, targetUserID string) error
}

type service struct {
	repo Repository
}

func NewService(repo Repository) Service {
	return &service{repo: repo}
}

func (s *service) Create(ctx context.Context, callerID string, parentID *string, assetType, title string, content *string) (*Asset, error) {
	if assetType != "note" && assetType != "folder" {
		return nil, ErrInvalidType
	}
	return s.repo.Create(ctx, callerID, parentID, assetType, title, content)
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
	if acl == nil {
		return nil, ErrForbidden
	}
	return existing, nil
}

func (s *service) Update(ctx context.Context, callerID, assetID, title string, content *string) (*Asset, error) {
	existing, err := s.repo.GetByID(ctx, assetID)
	if err != nil {
		return nil, err
	}
	if err := s.requireWriteAccess(ctx, existing, callerID); err != nil {
		return nil, err
	}
	return s.repo.Update(ctx, assetID, title, content)
}

func (s *service) Delete(ctx context.Context, callerID, assetID string) error {
	existing, err := s.repo.GetByID(ctx, assetID)
	if err != nil {
		return err
	}
	if err := s.requireWriteAccess(ctx, existing, callerID); err != nil {
		return err
	}
	return s.repo.Delete(ctx, assetID)
}

func (s *service) requireWriteAccess(ctx context.Context, asset *Asset, callerID string) error {
	if asset.OwnerID == callerID {
		return nil
	}
	acl, err := s.repo.GetACLEntry(ctx, asset.AssetID, callerID)
	if err != nil {
		return err
	}
	if acl == nil || acl.AccessLevel != "write" {
		return ErrForbidden
	}
	return nil
}

func (s *service) Share(ctx context.Context, callerID, assetID, targetUserID, accessLevel string) error {
	asset, err := s.repo.GetByID(ctx, assetID)
	if err != nil {
		return err
	}
	if err := s.requireWriteAccess(ctx, asset, callerID); err != nil {
		return err
	}
	if err := s.repo.UpsertACLEntry(ctx, assetID, targetUserID, accessLevel); err != nil {
		return err
	}
	if asset.Type == "folder" {
		descendants, err := s.repo.GetDescendantIDs(ctx, assetID)
		if err != nil {
			return err
		}
		for _, childID := range descendants {
			if err := s.repo.UpsertACLEntry(ctx, childID, targetUserID, accessLevel); err != nil {
				return err
			}
		}
	}
	return nil
}

func (s *service) RevokeShare(ctx context.Context, callerID, assetID, targetUserID string) error {
	asset, err := s.repo.GetByID(ctx, assetID)
	if err != nil {
		return err
	}
	if err := s.requireWriteAccess(ctx, asset, callerID); err != nil {
		return err
	}
	if err := s.repo.DeleteACLEntry(ctx, assetID, targetUserID); err != nil {
		return err
	}
	if asset.Type == "folder" {
		descendants, err := s.repo.GetDescendantIDs(ctx, assetID)
		if err != nil {
			return err
		}
		for _, childID := range descendants {
			if err := s.repo.DeleteACLEntry(ctx, childID, targetUserID); err != nil {
				return err
			}
		}
	}
	return nil
}
