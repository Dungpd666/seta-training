package team_test

import (
	"context"
	"errors"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/dungpd/seta/core-service/internal/team"
	"github.com/redis/go-redis/v9"
)

func newSvc() (team.Service, *mockTeamRepo) {
	repo := newMockTeamRepo()
	mr, _ := miniredis.Run()
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	return team.NewService(repo, rdb, &mockPublisher{}), repo
}

func TestCreateTeam_CreatorAutoAddedAsManager(t *testing.T) {
	svc, repo := newSvc()
	ctx := context.Background()

	result, err := svc.CreateTeam(ctx, "alice-id", "Alpha Team")
	if err != nil {
		t.Fatalf("CreateTeam: %v", err)
	}
	if result.TeamID == "" {
		t.Fatal("expected non-empty team_id")
	}
	if result.CreatedBy != "alice-id" {
		t.Errorf("created_by = %q, want %q", result.CreatedBy, "alice-id")
	}

	role, err := repo.GetMemberRole(ctx, result.TeamID, "alice-id")
	if err != nil {
		t.Fatalf("GetMemberRole: %v", err)
	}
	if role != "manager" {
		t.Errorf("creator role = %q, want manager", role)
	}
}

func TestAddMember_ManagerCanAdd(t *testing.T) {
	svc, _ := newSvc()
	ctx := context.Background()

	result, _ := svc.CreateTeam(ctx, "alice-id", "Alpha Team")

	err := svc.AddMember(ctx, result.TeamID, "alice-id", "bob-id")
	if err != nil {
		t.Errorf("manager should be able to add member, got: %v", err)
	}
}

func TestAddMember_NonMemberForbidden(t *testing.T) {
	svc, _ := newSvc()
	ctx := context.Background()

	result, _ := svc.CreateTeam(ctx, "alice-id", "Alpha Team")

	err := svc.AddMember(ctx, result.TeamID, "bob-id", "charlie-id")
	if !errors.Is(err, team.ErrNotTeamManager) {
		t.Errorf("expected ErrNotTeamManager, got: %v", err)
	}
}

func TestAddMember_MemberForbidden(t *testing.T) {
	svc, _ := newSvc()
	ctx := context.Background()

	result, _ := svc.CreateTeam(ctx, "alice-id", "Alpha Team")
	_ = svc.AddMember(ctx, result.TeamID, "alice-id", "bob-id") // bob là member

	err := svc.AddMember(ctx, result.TeamID, "bob-id", "charlie-id")
	if !errors.Is(err, team.ErrNotTeamManager) {
		t.Errorf("expected ErrNotTeamManager, got: %v", err)
	}
}

func TestRemoveMember_ManagerCanRemove(t *testing.T) {
	svc, _ := newSvc()
	ctx := context.Background()

	result, _ := svc.CreateTeam(ctx, "alice-id", "Alpha Team")
	_ = svc.AddMember(ctx, result.TeamID, "alice-id", "bob-id")

	err := svc.RemoveMember(ctx, result.TeamID, "alice-id", "bob-id")
	if err != nil {
		t.Errorf("manager should be able to remove member, got: %v", err)
	}
}

func TestRemoveMember_NonManagerForbidden(t *testing.T) {
	svc, _ := newSvc()
	ctx := context.Background()

	result, _ := svc.CreateTeam(ctx, "alice-id", "Alpha Team")
	_ = svc.AddMember(ctx, result.TeamID, "alice-id", "bob-id")

	err := svc.RemoveMember(ctx, result.TeamID, "bob-id", "alice-id")
	if !errors.Is(err, team.ErrNotTeamManager) {
		t.Errorf("expected ErrNotTeamManager, got: %v", err)
	}
}

func TestPromoteToManager_CreatorCanPromote(t *testing.T) {
	svc, repo := newSvc()
	ctx := context.Background()

	result, _ := svc.CreateTeam(ctx, "alice-id", "Alpha Team")
	_ = svc.AddMember(ctx, result.TeamID, "alice-id", "bob-id")

	err := svc.PromoteToManager(ctx, result.TeamID, "alice-id", "bob-id")
	if err != nil {
		t.Fatalf("creator should be able to promote, got: %v", err)
	}

	role, _ := repo.GetMemberRole(ctx, result.TeamID, "bob-id")
	if role != "manager" {
		t.Errorf("bob role = %q, want manager", role)
	}
}

func TestPromoteToManager_NonCreatorForbidden(t *testing.T) {
	svc, _ := newSvc()
	ctx := context.Background()

	result, _ := svc.CreateTeam(ctx, "alice-id", "Alpha Team")
	_ = svc.AddMember(ctx, result.TeamID, "alice-id", "bob-id")

	// bob là manager của team khác nhưng không phải creator của team này
	err := svc.PromoteToManager(ctx, result.TeamID, "bob-id", "charlie-id")
	if !errors.Is(err, team.ErrNotTeamCreator) {
		t.Errorf("expected ErrNotTeamCreator, got: %v", err)
	}
}

func TestDemoteFromManager_CreatorCanDemote(t *testing.T) {
	svc, repo := newSvc()
	ctx := context.Background()

	result, _ := svc.CreateTeam(ctx, "alice-id", "Alpha Team")
	_ = svc.AddMember(ctx, result.TeamID, "alice-id", "bob-id")
	_ = svc.PromoteToManager(ctx, result.TeamID, "alice-id", "bob-id")

	err := svc.DemoteFromManager(ctx, result.TeamID, "alice-id", "bob-id")
	if err != nil {
		t.Fatalf("creator should be able to demote, got: %v", err)
	}

	role, _ := repo.GetMemberRole(ctx, result.TeamID, "bob-id")
	if role != "member" {
		t.Errorf("bob role = %q, want member", role)
	}
}

func TestDemoteFromManager_CannotDemoteCreator(t *testing.T) {
	svc, _ := newSvc()
	ctx := context.Background()

	result, _ := svc.CreateTeam(ctx, "alice-id", "Alpha Team")

	err := svc.DemoteFromManager(ctx, result.TeamID, "alice-id", "alice-id")
	if !errors.Is(err, team.ErrCannotDemoteCreator) {
		t.Errorf("expected ErrCannotDemoteCreator, got: %v", err)
	}
}
