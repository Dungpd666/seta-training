package user

import (
	"net/http"

	"github.com/dungpd/seta/auth-service/internal/response"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	svc Service
}

func NewHandler(svc Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) ListUsers(c *gin.Context) {
	users, err := h.svc.ListAll(c.Request.Context())
	if err != nil {
		response.Error(c, http.StatusInternalServerError, response.ErrInternal, "failed to list users")
		return
	}

	result := make([]gin.H, len(users))
	for i, u := range users {
		result[i] = gin.H{
			"user_id":    u.UserID,
			"username":   u.Username,
			"email":      u.Email,
			"role":       u.Role,
			"created_at": u.CreatedAt,
		}
	}
	response.Success(c, result)
}
