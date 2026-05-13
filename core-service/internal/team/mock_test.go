package team_test

import (
	"context"
	"fmt"
	"strings"

	"github.com/dungpd/seta/core-service/internal/team"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type mockPublisher struct{}

func (m *mockPublisher) Publish(_ context.Context, _ string, _ any) error { return nil }

type mockTeamRepo struct {
	teams   map[string]*team.Team
	members map[string]string // "teamID:userID" → role
}

func newMockTeamRepo() *mockTeamRepo {
	return &mockTeamRepo{
		teams:   make(map[string]*team.Team),
		members: make(map[string]string),
	}
}

func (m *mockTeamRepo) Create(_ context.Context, teamName, createdBy string) (*team.Team, error) {
	t := &team.Team{
		TeamID:    uuid.NewString(),
		TeamName:  teamName,
		CreatedBy: createdBy,
	}
	m.teams[t.TeamID] = t
	return t, nil
}

func (m *mockTeamRepo) GetByID(_ context.Context, teamID string) (*team.Team, error) {
	t, ok := m.teams[teamID]
	if !ok {
		return nil, pgx.ErrNoRows
	}
	return t, nil
}

func (m *mockTeamRepo) AddMember(_ context.Context, teamID, userID, role string) error {
	m.members[fmt.Sprintf("%s:%s", teamID, userID)] = role
	return nil
}

func (m *mockTeamRepo) RemoveMember(_ context.Context, teamID, userID string) error {
	delete(m.members, fmt.Sprintf("%s:%s", teamID, userID))
	return nil
}

func (m *mockTeamRepo) GetMemberRole(_ context.Context, teamID, userID string) (string, error) {
	role, ok := m.members[fmt.Sprintf("%s:%s", teamID, userID)]
	if !ok {
		return "", pgx.ErrNoRows
	}
	return role, nil
}

func (m *mockTeamRepo) GetUserByID(_ context.Context, userID string) (*team.UserProjection, error) {
	return &team.UserProjection{UserID: userID}, nil
}

func (m *mockTeamRepo) ListMembers(_ context.Context, teamID string) ([]*team.TeamMember, error) {
	prefix := teamID + ":"
	var members []*team.TeamMember
	for key, role := range m.members {
		if strings.HasPrefix(key, prefix) {
			members = append(members, &team.TeamMember{UserID: key[len(prefix):], Role: role})
		}
	}
	return members, nil
}
