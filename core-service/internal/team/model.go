package team

import "errors"

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

const (
	EventUserCreated = "USER_CREATED"
	EventUserUpdated = "USER_UPDATED"
	EventUserDeleted = "USER_DELETED"
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
