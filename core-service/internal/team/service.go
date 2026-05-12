package team

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/dungpd/seta/core-service/internal/cache"
	"github.com/jackc/pgx/v5"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog/log"
)

type Service interface {
	CreateTeam(ctx context.Context, createdBy, teamName string) (*Team, error)
	AddMember(ctx context.Context, teamID, callerID, targetUserID string) error
	RemoveMember(ctx context.Context, teamID, callerID, targetUserID string) error
	PromoteToManager(ctx context.Context, teamID, callerID, targetUserID string) error
	DemoteFromManager(ctx context.Context, teamID, callerID, targetUserID string) error
	GetMembers(ctx context.Context, teamID, callerID string) ([]*TeamMember, error)
}

type service struct {
	repo      TeamRepository
	rdb       *redis.Client
	publisher Publisher
}

func NewService(repo TeamRepository, rdb *redis.Client, publisher Publisher) Service {
	return &service{
		repo:      repo,
		rdb:       rdb,
		publisher: publisher,
	}
}

func (s *service) CreateTeam(ctx context.Context, createdBy, teamName string) (*Team, error) {
	team, err := s.repo.Create(ctx, teamName, createdBy)
	if err != nil {
		return nil, err
	}
	if err := s.repo.AddMember(ctx, team.TeamID, createdBy, RoleManager); err != nil {
		return nil, err
	}
	s.publishEvent(ctx, EventTeamCreated, team.TeamID, createdBy)
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
	if err := s.repo.AddMember(ctx, teamID, targetUserID, RoleMember); err != nil {
		return err
	}
	s.publishEvent(ctx, EventMemberAdded, teamID, targetUserID)
	s.rdb.Del(ctx, cache.TeamMembersKey(teamID))
	return nil
}

func (s *service) RemoveMember(ctx context.Context, teamID, callerID, targetUserID string) error {
	if err := s.requireTeamManager(ctx, teamID, callerID); err != nil {
		return err
	}
	if err := s.repo.RemoveMember(ctx, teamID, targetUserID); err != nil {
		return err
	}
	s.publishEvent(ctx, EventMemberRemoved, teamID, targetUserID)
	s.rdb.Del(ctx, cache.TeamMembersKey(teamID))
	return nil
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
	if err := s.repo.AddMember(ctx, teamID, targetUserID, RoleManager); err != nil {
		return err
	}
	s.publishEvent(ctx, EventManagerAdded, teamID, targetUserID)
	return nil
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
		return ErrNotTeamManager
	}
	if err := s.repo.AddMember(ctx, teamID, targetUserID, RoleMember); err != nil {
		return err
	}
	s.publishEvent(ctx, EventManagerRemoved, teamID, targetUserID)
	return nil
}

func (s *service) GetMembers(ctx context.Context, teamID, callerID string) ([]*TeamMember, error) {
	_, err := s.repo.GetMemberRole(ctx, teamID, callerID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotTeamMember
		}
		return nil, err
	}

	cacheKey := cache.TeamMembersKey(teamID)

	cached, err := s.rdb.Get(ctx, cacheKey).Result()
	if err == nil {
		var members []*TeamMember
		if err := json.Unmarshal([]byte(cached), &members); err == nil {
			return members, nil
		}
	}

	members, err := s.repo.ListMembers(ctx, teamID)
	if err != nil {
		return nil, err
	}

	data, err := json.Marshal(members)
	if err != nil {
		log.Warn().Err(err).Str("team_id", teamID).Msg("failed to marshal team members for cache")
	} else {
		s.rdb.Set(ctx, cacheKey, data, 5*time.Minute)
	}

	return members, nil
}

func (s *service) publishEvent(ctx context.Context, evt, teamID, userID string) {
	if err := s.publisher.Publish(ctx, TopicTeamActivity, TeamEvent{
		Event:  evt,
		TeamID: teamID,
		UserID: userID,
	}); err != nil {
		log.Error().Err(err).Str("event", evt).Str("team_id", teamID).Msg("failed to publish team event")
	}
}

func (s *service) mustGetTeam(ctx context.Context, teamID string) (*Team, error) {
	team, err := s.repo.GetByID(ctx, teamID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrTeamNotFound
	}
	return team, err
}

func (s *service) requireTeamManager(ctx context.Context, teamID, callerID string) error {
	if _, err := s.mustGetTeam(ctx, teamID); err != nil {
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
	team, err := s.mustGetTeam(ctx, teamID)
	if err != nil {
		return err
	}
	if team.CreatedBy != callerID {
		return ErrNotTeamCreator
	}
	return nil
}
