package team

import "context"

type Service interface {
	CreateTeam(ctx context.Context, createdBy, teamName string) (*Team, error)
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
