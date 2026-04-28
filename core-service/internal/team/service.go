package team

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
)

var ErrNotTeamManager = errors.New("user is not a team manager")

type Service interface {
	CreateTeam(ctx context.Context, createdBy, teamName string) (*Team, error)
	AddMember(ctx context.Context, teamID, callerID, targetUserID string) error
	RemoveMember(ctx context.Context, teamID, callerID, targetUserID string) error
}

type service struct {
	repo TeamRepository
}

func NewService(repo TeamRepository) Service {
	return &service{repo: repo}
}

func (s *service) CreateTeam(ctx context.Context, createdBy, teamName string) (*Team, error) {
	team, err := s.repo.Create(ctx, teamName, createdBy)
	if err != nil {
		return nil, err
	}
	if err := s.repo.AddMember(ctx, team.TeamID, createdBy, "manager"); err != nil {
		return nil, err
	}
	return team, nil
}

func (s *service) AddMember(ctx context.Context, teamID, callerID, targetUserID string) error {
	if err := s.requireTeamManager(ctx, teamID, callerID); err != nil {
		return err
	}
	return s.repo.AddMember(ctx, teamID, targetUserID, "member")
}

func (s *service) RemoveMember(ctx context.Context, teamID, callerID, targetUserID string) error {
	if err := s.requireTeamManager(ctx, teamID, callerID); err != nil {
		return err
	}
	return s.repo.RemoveMember(ctx, teamID, targetUserID)
}

func (s *service) requireTeamManager(ctx context.Context, teamID, callerID string) error {
	role, err := s.repo.GetMemberRole(ctx, teamID, callerID)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotTeamManager
	}
	if err != nil {
		return err
	}
	if role != "manager" {
		return ErrNotTeamManager
	}
	return nil
}
