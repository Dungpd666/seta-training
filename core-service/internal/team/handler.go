package team

import (
	"errors"
	"net/http"

	"github.com/dungpd/seta/core-service/internal/middleware"
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

	role, ok := middleware.CallerRole(c)
	if !ok || role != RoleManager {
		response.Error(c, http.StatusForbidden, response.ErrForbidden, "only manager can create team")
		return
	}

	callerID, ok := middleware.CallerID(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, response.ErrUnauthorized, "missing caller")
		return
	}
	team, err := h.svc.CreateTeam(c.Request.Context(), callerID, body.TeamName)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, response.ErrInternal, err.Error())
		return
	}

	response.SuccessWithStatus(c, http.StatusCreated, team)
}

func (h *Handler) AddMember(c *gin.Context) {
	teamID := c.Param("id")
	callerID, ok := middleware.CallerID(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, response.ErrUnauthorized, "missing caller")
		return
	}

	var body struct {
		UserID string `json:"user_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		response.Error(c, http.StatusBadRequest, response.ErrBadRequest, "user_id is required")
		return
	}

	err := h.svc.AddMember(c.Request.Context(), teamID, callerID, body.UserID)
	if errors.Is(err, ErrNotTeamManager) {
		response.Error(c, http.StatusForbidden, response.ErrForbidden, err.Error())
		return
	}
	if errors.Is(err, ErrTeamNotFound) {
		response.Error(c, http.StatusNotFound, response.ErrNotFound, err.Error())
		return
	}
	if errors.Is(err, ErrUserNotFound) {
		response.Error(c, http.StatusUnprocessableEntity, response.ErrBadRequest, err.Error())
		return
	}
	if err != nil {
		response.Error(c, http.StatusInternalServerError, response.ErrInternal, err.Error())
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) RemoveMember(c *gin.Context) {
	teamID := c.Param("id")
	targetUserID := c.Param("userId")
	callerID, ok := middleware.CallerID(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, response.ErrUnauthorized, "missing caller")
		return
	}

	err := h.svc.RemoveMember(c.Request.Context(), teamID, callerID, targetUserID)
	if errors.Is(err, ErrNotTeamManager) {
		response.Error(c, http.StatusForbidden, response.ErrForbidden, err.Error())
		return
	}
	if errors.Is(err, ErrTeamNotFound) {
		response.Error(c, http.StatusNotFound, response.ErrNotFound, err.Error())
		return
	}
	if err != nil {
		response.Error(c, http.StatusInternalServerError, response.ErrInternal, err.Error())
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) AddManager(c *gin.Context) {
	teamID := c.Param("id")
	callerID, ok := middleware.CallerID(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, response.ErrUnauthorized, "missing caller")
		return
	}

	var body struct {
		UserID string `json:"user_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		response.Error(c, http.StatusBadRequest, response.ErrBadRequest, "user_id is required")
		return
	}

	err := h.svc.PromoteToManager(c.Request.Context(), teamID, callerID, body.UserID)
	if errors.Is(err, ErrNotTeamCreator) {
		response.Error(c, http.StatusForbidden, response.ErrForbidden, err.Error())
		return
	}
	if errors.Is(err, ErrTeamNotFound) {
		response.Error(c, http.StatusNotFound, response.ErrNotFound, err.Error())
		return
	}
	if errors.Is(err, ErrUserNotFound) {
		response.Error(c, http.StatusUnprocessableEntity, response.ErrBadRequest, err.Error())
		return
	}
	if err != nil {
		response.Error(c, http.StatusInternalServerError, response.ErrInternal, err.Error())
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) RemoveManager(c *gin.Context) {
	teamID := c.Param("id")
	targetUserID := c.Param("userId")
	callerID, ok := middleware.CallerID(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, response.ErrUnauthorized, "missing caller")
		return
	}

	err := h.svc.DemoteFromManager(c.Request.Context(), teamID, callerID, targetUserID)
	if errors.Is(err, ErrNotTeamCreator) || errors.Is(err, ErrCannotDemoteCreator) {
		response.Error(c, http.StatusForbidden, response.ErrForbidden, err.Error())
		return
	}
	if errors.Is(err, ErrTeamNotFound) {
		response.Error(c, http.StatusNotFound, response.ErrNotFound, err.Error())
		return
	}
	if errors.Is(err, ErrNotTeamMember) {
		response.Error(c, http.StatusUnprocessableEntity, response.ErrBadRequest, err.Error())
		return
	}
	if err != nil {
		response.Error(c, http.StatusInternalServerError, response.ErrInternal, err.Error())
		return
	}
	c.Status(http.StatusNoContent)
}
