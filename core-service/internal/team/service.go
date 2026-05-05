package team

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
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

func NewTeamService(repo TeamRepository) Service {
	return &service{repo: repo}
}

func (s *service) CreateTeam(ctx context.Context, createdBy, teamName string) (*Team, error) {
	team, err := s.repo.Create(ctx, teamName, createdBy)
	if err != nil {
		return nil, err
	}
	if err := s.repo.AddMember(ctx, team.TeamID, createdBy, RoleManager); err != nil {
		return nil, err
	}
	return team, nil
}

func (s *service) AddMember(ctx context.Context, teamID, callerID, targetUserID string) error {
	if err := s.requireTeamManager(ctx, teamID, callerID); err != nil {
		return err
	}
	if _, err := s.repo.GetMemberRole(ctx, teamID, targetUserID); err == nil {
		return ErrAlreadyMember
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return err
	}
	if _, err := s.repo.GetUserByID(ctx, targetUserID); errors.Is(err, pgx.ErrNoRows) {
		return ErrUserNotFound
	} else if err != nil {
		return err
	}
	return s.repo.AddMember(ctx, teamID, targetUserID, RoleMember)
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
	if _, err := s.repo.GetUserByID(ctx, targetUserID); errors.Is(err, pgx.ErrNoRows) {
		return ErrUserNotFound
	} else if err != nil {
		return err
	}
	return s.repo.AddMember(ctx, teamID, targetUserID, RoleManager)
}

func (s *service) DemoteFromManager(ctx context.Context, teamID, callerID, targetUserID string) error {
	if err := s.requireTeamCreator(ctx, teamID, callerID); err != nil {
		return err
	}
	if callerID == targetUserID {
		return ErrCannotDemoteCreator
	}
	role, err := s.repo.GetMemberRole(ctx, teamID, targetUserID)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotTeamMember
	}
	if err != nil {
		return err
	}
	if role != RoleManager {
		return ErrNotTeamMember
	}
	return s.repo.AddMember(ctx, teamID, targetUserID, RoleMember)
}

func (s *service) requireTeamManager(ctx context.Context, teamID, callerID string) error {
	if _, err := s.repo.GetByID(ctx, teamID); errors.Is(err, pgx.ErrNoRows) {
		return ErrTeamNotFound
	} else if err != nil {
		return err
	}
	role, err := s.repo.GetMemberRole(ctx, teamID, callerID)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotTeamManager
	}
	if err != nil {
		return err
	}
	if role != RoleManager {
		return ErrNotTeamManager
	}
	return nil
}

func (s *service) requireTeamCreator(ctx context.Context, teamID, callerID string) error {
	team, err := s.repo.GetByID(ctx, teamID)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrTeamNotFound
	}
	if err != nil {
		return err
	}
	if team.CreatedBy != callerID {
		return ErrNotTeamCreator
	}
	return nil
}
