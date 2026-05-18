package asset_test

import (
	"context"
	"strings"

	"github.com/dungpd/seta/core-service/internal/asset"
	"github.com/google/uuid"
)

type mockPublisher struct{}

func (m *mockPublisher) Publish(_ context.Context, _ string, _ any) error { return nil }

type upsertCall struct {
	assetID, userID, accessLevel string
}

type mockAssetRepo struct {
	assets      map[string]*asset.Asset
	acls        map[string]*asset.AssetACL
	descendants map[string][]string
	managers    map[string]bool
	users       map[string]bool
	upsertCalls []upsertCall
}

func newMockAssetRepo() *mockAssetRepo {
	return &mockAssetRepo{
		assets:      make(map[string]*asset.Asset),
		acls:        make(map[string]*asset.AssetACL),
		descendants: make(map[string][]string),
		managers:    make(map[string]bool),
		users:       make(map[string]bool),
	}
}

func (m *mockAssetRepo) Create(_ context.Context, ownerID string, parentID *string, assetType, title string, content *string) (*asset.Asset, error) {
	a := &asset.Asset{
		AssetID:  uuid.NewString(),
		OwnerID:  ownerID,
		ParentID: parentID,
		Type:     assetType,
		Title:    title,
		Content:  content,
	}
	m.assets[a.AssetID] = a
	return a, nil
}

func (m *mockAssetRepo) GetByID(_ context.Context, assetID string) (*asset.Asset, error) {
	a, ok := m.assets[assetID]
	if !ok {
		return nil, asset.ErrNotFound
	}
	return a, nil
}

func (m *mockAssetRepo) Update(_ context.Context, assetID, title string, content *string) (*asset.Asset, error) {
	a, ok := m.assets[assetID]
	if !ok {
		return nil, asset.ErrNotFound
	}
	a.Title = title
	a.Content = content
	return a, nil
}

func (m *mockAssetRepo) Delete(_ context.Context, assetID string) error {
	delete(m.assets, assetID)
	return nil
}

func (m *mockAssetRepo) GetACLEntry(_ context.Context, assetID, userID string) (*asset.AssetACL, error) {
	return m.acls[assetID+":"+userID], nil
}

func (m *mockAssetRepo) UpsertACLEntry(_ context.Context, assetID, userID, accessLevel string) error {
	m.upsertCalls = append(m.upsertCalls, upsertCall{assetID, userID, accessLevel})
	m.acls[assetID+":"+userID] = &asset.AssetACL{AssetID: assetID, UserID: userID, AccessLevel: accessLevel}
	return nil
}

func (m *mockAssetRepo) DeleteACLEntry(_ context.Context, assetID, userID string) error {
	delete(m.acls, assetID+":"+userID)
	return nil
}

func (m *mockAssetRepo) UpsertACLWithCascade(_ context.Context, assetID, assetType, userID, accessLevel string) ([]string, error) {
	m.upsertCalls = append(m.upsertCalls, upsertCall{assetID, userID, accessLevel})
	m.acls[assetID+":"+userID] = &asset.AssetACL{AssetID: assetID, UserID: userID, AccessLevel: accessLevel}
	var descendants []string
	if assetType == asset.AssetTypeFolder {
		descendants = m.descendants[assetID]
		for _, id := range descendants {
			m.upsertCalls = append(m.upsertCalls, upsertCall{id, userID, accessLevel})
			m.acls[id+":"+userID] = &asset.AssetACL{AssetID: id, UserID: userID, AccessLevel: accessLevel}
		}
	}
	return descendants, nil
}

func (m *mockAssetRepo) DeleteACLWithCascade(_ context.Context, assetID, assetType, userID string) ([]string, error) {
	delete(m.acls, assetID+":"+userID)
	var descendants []string
	if assetType == asset.AssetTypeFolder {
		descendants = m.descendants[assetID]
		for _, id := range descendants {
			delete(m.acls, id+":"+userID)
		}
	}
	return descendants, nil
}

func (m *mockAssetRepo) GetDescendantIDs(_ context.Context, assetID string) ([]string, error) {
	return m.descendants[assetID], nil
}

func (m *mockAssetRepo) IsManagerOfOwner(_ context.Context, callerID, ownerID string) (bool, error) {
	return m.managers[callerID+":"+ownerID], nil
}

func (m *mockAssetRepo) UserExists(_ context.Context, userID string) (bool, error) {
	return m.users[userID], nil
}

func (m *mockAssetRepo) ListACLByAsset(_ context.Context, assetID string) ([]*asset.AssetACL, error) {
	prefix := assetID + ":"
	var entries []*asset.AssetACL
	for key, acl := range m.acls {
		if strings.HasPrefix(key, prefix) {
			entries = append(entries, acl)
		}
	}
	return entries, nil
}

func (m *mockAssetRepo) List(_ context.Context, ownerID string, limit, offset int32) ([]*asset.Asset, error) {
	var result []*asset.Asset
	for _, a := range m.assets {
		if a.OwnerID == ownerID {
			result = append(result, a)
		}
	}
	start := int(offset)
	if start >= len(result) {
		return []*asset.Asset{}, nil
	}
	end := start + int(limit)
	if end > len(result) {
		end = len(result)
	}
	return result[start:end], nil
}

func (m *mockAssetRepo) CountByOwner(_ context.Context, ownerID string) (int64, error) {
	var count int64
	for _, a := range m.assets {
		if a.OwnerID == ownerID {
			count++
		}
	}
	return count, nil
}
