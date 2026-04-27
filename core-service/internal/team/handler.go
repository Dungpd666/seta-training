package team

import (
	"net/http"

	"github.com/dungpd/seta/core-service/internal/response"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	svc Service
}

func NewHandler(svc Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) CreateTeam(c *gin.Context) {
	var body struct {
		TeamName string `json:"team_name" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		response.Error(c, http.StatusBadRequest, response.ErrBadRequest, "team_name is required")
		return
	}

	if role, _ := c.Get("role"); role != "manager" {
		response.Error(c, http.StatusForbidden, "FORBIDDEN", "only manager can create team")
		return
	}

	userID, _ := c.Get("user_id")
	team, err := h.svc.CreateTeam(c.Request.Context(), userID.(string), body.TeamName)
	if err != nil {
		response.Error(c, http.StatusBadRequest, response.ErrBadRequest, err.Error())
		return
	}

	response.SuccessWithStatus(c, http.StatusCreated, team)
}
