package team

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
)

var (
	ErrNotTeamManager      = errors.New("user is not a team manager")
	ErrNotTeamCreator      = errors.New("user is not the team creator")
	ErrCannotDemoteCreator = errors.New("cannot demote team creator")
)

type Service interface {
	CreateTeam(ctx context.Context, createdBy, teamName string) (*Team, error)
	AddMember(ctx context.Context, teamID, callerID, targetUserID string) error
	RemoveMember(ctx context.Context, teamID, callerID, targetUserID string) error
	PromoteToManager(ctx context.Context, teamID, callerID, targetUserID string) error
	DemoteFromManager(ctx context.Context, teamID, callerID, targetUserID string) error
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

func (s *service) PromoteToManager(ctx context.Context, teamID, callerID, targetUserID string) error {
	if err := s.requireTeamCreator(ctx, teamID, callerID); err != nil {
		return err
	}
	return s.repo.AddMember(ctx, teamID, targetUserID, "manager")
}

func (s *service) DemoteFromManager(ctx context.Context, teamID, callerID, targetUserID string) error {
	if err := s.requireTeamCreator(ctx, teamID, callerID); err != nil {
		return err
	}
	if callerID == targetUserID {
		return ErrCannotDemoteCreator
	}
	return s.repo.AddMember(ctx, teamID, targetUserID, "member")
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

func (s *service) requireTeamCreator(ctx context.Context, teamID, callerID string) error {
	team, err := s.repo.GetByID(ctx, teamID)
	if errors.Is(err, pgx.ErrNoRows) {
		return errors.New("team not found")
	}
	if err != nil {
		return err
	}
	if team.CreatedBy != callerID {
		return ErrNotTeamCreator
	}
	return nil
}
