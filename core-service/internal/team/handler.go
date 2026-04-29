package team

import (
	"errors"
	"net/http"

	"github.com/dungpd/seta/core-service/internal/response"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	svc Service
}

func NewTeamHandler(svc Service) *Handler {
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

func (h *Handler) AddMember(c *gin.Context) {
	teamID := c.Param("id")
	callerID, _ := c.Get("user_id")

	var body struct {
		UserID string `json:"user_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		response.Error(c, http.StatusBadRequest, response.ErrBadRequest, "user_id is required")
		return
	}

	err := h.svc.AddMember(c.Request.Context(), teamID, callerID.(string), body.UserID)
	if errors.Is(err, ErrNotTeamManager) {
		response.Error(c, http.StatusForbidden, "FORBIDDEN", err.Error())
		return
	}
	if err != nil {
		response.Error(c, http.StatusBadRequest, response.ErrBadRequest, err.Error())
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) RemoveMember(c *gin.Context) {
	teamID := c.Param("id")
	targetUserID := c.Param("userId")
	callerID, _ := c.Get("user_id")

	err := h.svc.RemoveMember(c.Request.Context(), teamID, callerID.(string), targetUserID)
	if errors.Is(err, ErrNotTeamManager) {
		response.Error(c, http.StatusForbidden, "FORBIDDEN", err.Error())
		return
	}
	if err != nil {
		response.Error(c, http.StatusBadRequest, response.ErrBadRequest, err.Error())
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) AddManager(c *gin.Context) {
	teamID := c.Param("id")
	callerID, _ := c.Get("user_id")

	var body struct {
		UserID string `json:"user_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		response.Error(c, http.StatusBadRequest, response.ErrBadRequest, "user_id is required")
		return
	}

	err := h.svc.PromoteToManager(c.Request.Context(), teamID, callerID.(string), body.UserID)
	if errors.Is(err, ErrNotTeamCreator) {
		response.Error(c, http.StatusForbidden, "FORBIDDEN", err.Error())
		return
	}
	if err != nil {
		response.Error(c, http.StatusBadRequest, response.ErrBadRequest, err.Error())
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) RemoveManager(c *gin.Context) {
	teamID := c.Param("id")
	targetUserID := c.Param("userId")
	callerID, _ := c.Get("user_id")

	err := h.svc.DemoteFromManager(c.Request.Context(), teamID, callerID.(string), targetUserID)
	if errors.Is(err, ErrNotTeamCreator) || errors.Is(err, ErrCannotDemoteCreator) {
		response.Error(c, http.StatusForbidden, "FORBIDDEN", err.Error())
		return
	}
	if err != nil {
		response.Error(c, http.StatusBadRequest, response.ErrBadRequest, err.Error())
		return
	}
	c.Status(http.StatusNoContent)
}
