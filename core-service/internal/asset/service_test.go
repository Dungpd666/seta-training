package asset_test

import (
	"context"
	"errors"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/dungpd/seta/core-service/internal/asset"
	"github.com/redis/go-redis/v9"
)

func newSvc() (asset.Service, *mockAssetRepo) {
	repo := newMockAssetRepo()
	mr, _ := miniredis.Run()
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	return asset.NewService(repo, rdb, &mockPublisher{}), repo
}

func TestShare_FolderCascadesToChildNotes(t *testing.T) {
	svc, repo := newSvc()
	ctx := context.Background()

	repo.assets["folder-1"] = &asset.Asset{AssetID: "folder-1", OwnerID: "alice", Type: asset.AssetTypeFolder, Title: "My Folder"}
	repo.assets["note-1"] = &asset.Asset{AssetID: "note-1", OwnerID: "alice", Type: asset.AssetTypeNote, Title: "Note 1"}
	repo.assets["note-2"] = &asset.Asset{AssetID: "note-2", OwnerID: "alice", Type: asset.AssetTypeNote, Title: "Note 2"}
	repo.descendants["folder-1"] = []string{"note-1", "note-2"}
	repo.users["bob"] = true

	if err := svc.Share(ctx, "alice", "folder-1", "bob", asset.AccessLevelRead); err != nil {
		t.Fatalf("Share: %v", err)
	}

	if len(repo.upsertCalls) != 3 {
		t.Fatalf("expected 3 UpsertACLEntry calls (folder + 2 notes), got %d", len(repo.upsertCalls))
	}

	seen := make(map[string]bool)
	for _, c := range repo.upsertCalls {
		seen[c.assetID] = true
	}
	for _, id := range []string{"folder-1", "note-1", "note-2"} {
		if !seen[id] {
			t.Errorf("missing UpsertACLEntry for assetID %q", id)
		}
	}
}

func TestGetByID_ManagerCanReadMemberAsset(t *testing.T) {
	svc, repo := newSvc()
	ctx := context.Background()

	repo.assets["asset-1"] = &asset.Asset{AssetID: "asset-1", OwnerID: "bob", Type: asset.AssetTypeNote, Title: "Bob's note"}
	repo.managers["alice:bob"] = true

	got, err := svc.GetByID(ctx, "alice", "asset-1")
	if err != nil {
		t.Fatalf("manager should read member asset, got: %v", err)
	}
	if got.AssetID != "asset-1" {
		t.Errorf("got AssetID %q, want asset-1", got.AssetID)
	}
}

func TestGetByID_NonManagerWithoutACLDenied(t *testing.T) {
	svc, repo := newSvc()
	ctx := context.Background()

	repo.assets["asset-1"] = &asset.Asset{AssetID: "asset-1", OwnerID: "bob", Type: asset.AssetTypeNote, Title: "Bob's note"}
	_ = repo // managers["charlie:bob"] is false by zero value

	_, err := svc.GetByID(ctx, "charlie", "asset-1")
	if !errors.Is(err, asset.ErrForbidden) {
		t.Errorf("expected ErrForbidden, got: %v", err)
	}
}
