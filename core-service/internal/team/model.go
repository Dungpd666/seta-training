package team

import (
	"context"
	"errors"
)

var (
	ErrNotTeamManager      = errors.New("user is not a team manager")
	ErrNotTeamCreator      = errors.New("user is not the team creator")
	ErrCannotDemoteCreator = errors.New("cannot demote team creator")
	ErrTeamNotFound        = errors.New("team not found")
	ErrUserNotFound        = errors.New("user not found")
	ErrNotTeamMember       = errors.New("user is not a team member")
	ErrAlreadyMember       = errors.New("user already a member")
)

const (
	RoleManager = "manager"
	RoleMember  = "member"
)

const TopicTeamActivity = "team.activity"

const (
	EventUserCreated    = "USER_CREATED"
	EventUserUpdated    = "USER_UPDATED"
	EventUserDeleted    = "USER_DELETED"
	EventTeamCreated    = "TEAM_CREATED"
	EventMemberAdded    = "MEMBER_ADDED"
	EventMemberRemoved  = "MEMBER_REMOVED"
	EventManagerAdded   = "MANAGER_ADDED"
	EventManagerRemoved = "MANAGER_REMOVED"
)

type UserEvent struct {
	Type     string `json:"type"`
	UserID   string `json:"user_id"`
	UserName string `json:"user_name"`
	Email    string `json:"email"`
	Role     string `json:"role"`
}

type UserProjection struct {
	UserID   string
	Username string
	Email    string
	Role     string
}

type Team struct {
	TeamID    string `json:"team_id"`
	TeamName  string `json:"team_name"`
	CreatedBy string `json:"created_by"`
}

type TeamEvent struct {
	Event  string `json:"event"`
	TeamID string `json:"team_id"`
	UserID string `json:"user_id,omitempty"`
}

type Publisher interface {
	Publish(ctx context.Context, topic string, payload any) error
}
