package team

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
