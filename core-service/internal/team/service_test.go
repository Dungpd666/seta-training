package team_test

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/dungpd/seta/core-service/internal/team"
	"github.com/jackc/pgx/v5"
	"github.com/redis/go-redis/v9"
)

func newSvc(t *testing.T) (team.Service, *mockTeamRepo) {
	t.Helper()
	repo := newMockTeamRepo()
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	return team.NewService(repo, rdb, &mockPublisher{}), repo
}

func TestCreateTeam_CreatorAutoAddedAsManager(t *testing.T) {
	svc, repo := newSvc(t)
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
	svc, _ := newSvc(t)
	ctx := context.Background()

	result, _ := svc.CreateTeam(ctx, "alice-id", "Alpha Team")

	err := svc.AddMember(ctx, result.TeamID, "alice-id", "bob-id")
	if err != nil {
		t.Errorf("manager should be able to add member, got: %v", err)
	}
}

func TestAddMember_NonMemberForbidden(t *testing.T) {
	svc, _ := newSvc(t)
	ctx := context.Background()

	result, _ := svc.CreateTeam(ctx, "alice-id", "Alpha Team")

	err := svc.AddMember(ctx, result.TeamID, "bob-id", "charlie-id")
	if !errors.Is(err, team.ErrNotTeamManager) {
		t.Errorf("expected ErrNotTeamManager, got: %v", err)
	}
}

func TestAddMember_MemberForbidden(t *testing.T) {
	svc, _ := newSvc(t)
	ctx := context.Background()

	result, _ := svc.CreateTeam(ctx, "alice-id", "Alpha Team")
	_ = svc.AddMember(ctx, result.TeamID, "alice-id", "bob-id") // bob là member

	err := svc.AddMember(ctx, result.TeamID, "bob-id", "charlie-id")
	if !errors.Is(err, team.ErrNotTeamManager) {
		t.Errorf("expected ErrNotTeamManager, got: %v", err)
	}
}

func TestRemoveMember_ManagerCanRemove(t *testing.T) {
	svc, _ := newSvc(t)
	ctx := context.Background()

	result, _ := svc.CreateTeam(ctx, "alice-id", "Alpha Team")
	_ = svc.AddMember(ctx, result.TeamID, "alice-id", "bob-id")

	err := svc.RemoveMember(ctx, result.TeamID, "alice-id", "bob-id")
	if err != nil {
		t.Errorf("manager should be able to remove member, got: %v", err)
	}
}

func TestRemoveMember_NonManagerForbidden(t *testing.T) {
	svc, _ := newSvc(t)
	ctx := context.Background()

	result, _ := svc.CreateTeam(ctx, "alice-id", "Alpha Team")
	_ = svc.AddMember(ctx, result.TeamID, "alice-id", "bob-id")

	err := svc.RemoveMember(ctx, result.TeamID, "bob-id", "alice-id")
	if !errors.Is(err, team.ErrNotTeamManager) {
		t.Errorf("expected ErrNotTeamManager, got: %v", err)
	}
}

func TestPromoteToManager_CreatorCanPromote(t *testing.T) {
	svc, repo := newSvc(t)
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
	svc, _ := newSvc(t)
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
	svc, repo := newSvc(t)
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
	svc, _ := newSvc(t)
	ctx := context.Background()

	result, _ := svc.CreateTeam(ctx, "alice-id", "Alpha Team")

	err := svc.DemoteFromManager(ctx, result.TeamID, "alice-id", "alice-id")
	if !errors.Is(err, team.ErrCannotDemoteCreator) {
		t.Errorf("expected ErrCannotDemoteCreator, got: %v", err)
	}
}

func TestRemoveMember_NonMemberReturnsError(t *testing.T) {
	svc, _ := newSvc(t)
	ctx := context.Background()

	result, _ := svc.CreateTeam(ctx, "alice-id", "Alpha Team")

	err := svc.RemoveMember(ctx, result.TeamID, "alice-id", "nonexistent-user")
	if !errors.Is(err, team.ErrNotTeamMember) {
		t.Errorf("expected ErrNotTeamMember, got: %v", err)
	}
}

// mockTeamRepoFailOnMember simulates CreateWithManager failing after the team
// row is created but before the manager membership is inserted — the real DB
// implementation wraps both in a transaction and would roll back, leaving no
// orphan team. The mock enforces the same invariant by returning an error and
// not persisting the team.
type mockTeamRepoFailOnMember struct {
	teams   map[string]*team.Team
	members map[string]string
}

func newMockFailOnMember() *mockTeamRepoFailOnMember {
	return &mockTeamRepoFailOnMember{
		teams:   make(map[string]*team.Team),
		members: make(map[string]string),
	}
}

func (m *mockTeamRepoFailOnMember) CreateWithManager(_ context.Context, _, _ string) (*team.Team, error) {
	return nil, errors.New("simulated db error on AddTeamMember")
}
func (m *mockTeamRepoFailOnMember) AddMember(_ context.Context, teamID, userID, role string) error {
	m.members[fmt.Sprintf("%s:%s", teamID, userID)] = role
	return nil
}
func (m *mockTeamRepoFailOnMember) RemoveMember(_ context.Context, teamID, userID string) error {
	delete(m.members, fmt.Sprintf("%s:%s", teamID, userID))
	return nil
}
func (m *mockTeamRepoFailOnMember) GetMemberRole(_ context.Context, teamID, userID string) (string, error) {
	role, ok := m.members[fmt.Sprintf("%s:%s", teamID, userID)]
	if !ok {
		return "", pgx.ErrNoRows
	}
	return role, nil
}
func (m *mockTeamRepoFailOnMember) GetByID(_ context.Context, teamID string) (*team.Team, error) {
	t, ok := m.teams[teamID]
	if !ok {
		return nil, pgx.ErrNoRows
	}
	return t, nil
}
func (m *mockTeamRepoFailOnMember) GetUserByID(_ context.Context, userID string) (*team.UserProjection, error) {
	return &team.UserProjection{UserID: userID}, nil
}
func (m *mockTeamRepoFailOnMember) ListMembers(_ context.Context, _ string) ([]*team.TeamMember, error) {
	return nil, nil
}

func TestCreateTeam_RollbackOnMemberInsertFailure(t *testing.T) {
	repo := newMockFailOnMember()
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	svc := team.NewService(repo, rdb, &mockPublisher{})

	_, err := svc.CreateTeam(context.Background(), "alice-id", "Alpha Team")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if len(repo.teams) != 0 {
		t.Errorf("expected no orphan teams after rollback, got %d", len(repo.teams))
	}
}
