package team

import (
	"errors"
	"net/http"

	"github.com/dungpd/seta/core-service/internal/middleware"
	"github.com/dungpd/seta/core-service/internal/response"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
)

type Handler struct {
	svc Service
}

func NewHandler(svc Service) *Handler {
	return &Handler{svc: svc}
}

func writeTeamErr(c *gin.Context, err error) bool {
	switch {
	case errors.Is(err, ErrNotTeamManager) || errors.Is(err, ErrNotTeamCreator) || errors.Is(err, ErrCannotDemoteCreator):
		response.Error(c, http.StatusForbidden, response.ErrForbidden, err.Error())
	case errors.Is(err, ErrTeamNotFound):
		response.Error(c, http.StatusNotFound, response.ErrNotFound, err.Error())
	case errors.Is(err, ErrUserNotFound) || errors.Is(err, ErrNotTeamMember) || errors.Is(err, ErrAlreadyMember):
		response.Error(c, http.StatusUnprocessableEntity, response.ErrUnprocessable, err.Error())
	case err != nil:
		log.Error().Err(err).Msg("internal error")
		response.Error(c, http.StatusInternalServerError, response.ErrInternal, "internal server error")
	default:
		return false
	}
	return true
}

func (h *Handler) CreateTeam(c *gin.Context) {
	var body CreateTeamRequest
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
	if writeTeamErr(c, err) {
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

	var body AddMemberRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		response.Error(c, http.StatusBadRequest, response.ErrBadRequest, "user_id is required")
		return
	}

	err := h.svc.AddMember(c.Request.Context(), teamID, callerID, body.UserID)
	if writeTeamErr(c, err) {
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
	if writeTeamErr(c, err) {
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

	var body AddManagerRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		response.Error(c, http.StatusBadRequest, response.ErrBadRequest, "user_id is required")
		return
	}

	err := h.svc.PromoteToManager(c.Request.Context(), teamID, callerID, body.UserID)
	if writeTeamErr(c, err) {
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
	if writeTeamErr(c, err) {
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) GetMembers(c *gin.Context) {
	teamID := c.Param("id")
	callerID, ok := middleware.CallerID(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, response.ErrUnauthorized, "missing caller")
		return
	}

	members, err := h.svc.GetMembers(c.Request.Context(), teamID, callerID)
	if writeTeamErr(c, err) {
		return
	}
	response.Success(c, members)
}
