package asset

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

func writeAssetErr(c *gin.Context, err error) bool {
	switch {
	case errors.Is(err, ErrNotFound) || errors.Is(err, ErrParentNotFound):
		response.Error(c, http.StatusNotFound, response.ErrNotFound, err.Error())
	case errors.Is(err, ErrForbidden):
		response.Error(c, http.StatusForbidden, response.ErrForbidden, err.Error())
	case errors.Is(err, ErrInvalidType) || errors.Is(err, ErrNoteRequiresParent) ||
		errors.Is(err, ErrParentNotFolder) || errors.Is(err, ErrFolderContentNotAllowed) ||
		errors.Is(err, ErrTargetUserNotFound):
		response.Error(c, http.StatusUnprocessableEntity, response.ErrUnprocessable, err.Error())
	case err != nil:
		log.Error().Err(err).Msg("internal error")
		response.Error(c, http.StatusInternalServerError, response.ErrInternal, "internal server error")
	default:
		return false
	}
	return true
}

func (h *Handler) Create(c *gin.Context) {
	var body struct {
		ParentID *string `json:"parent_id"`
		Type     string  `json:"type" binding:"required,oneof=folder note"`
		Title    string  `json:"title" binding:"required"`
		Content  *string `json:"content"`
	}

	if err := c.ShouldBindJSON(&body); err != nil {
		response.Error(c, http.StatusBadRequest, response.ErrBadRequest, err.Error())
		return
	}

	callerID, ok := middleware.CallerID(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, response.ErrUnauthorized, "missing caller")
		return
	}
	asset, err := h.svc.Create(c.Request.Context(), callerID, body.ParentID, body.Type, body.Title, body.Content)
	if writeAssetErr(c, err) {
		return
	}
	response.SuccessWithStatus(c, http.StatusCreated, asset)
}

func (h *Handler) GetByID(c *gin.Context) {
	assetID := c.Param("id")
	callerID, ok := middleware.CallerID(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, response.ErrUnauthorized, "missing caller")
		return
	}
	asset, err := h.svc.GetByID(c.Request.Context(), callerID, assetID)
	if writeAssetErr(c, err) {
		return
	}
	response.Success(c, asset)
}

func (h *Handler) Update(c *gin.Context) {
	assetID := c.Param("id")
	var body struct {
		Title   string  `json:"title" binding:"required"`
		Content *string `json:"content"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		response.Error(c, http.StatusBadRequest, response.ErrBadRequest, err.Error())
		return
	}

	callerID, ok := middleware.CallerID(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, response.ErrUnauthorized, "missing caller")
		return
	}
	asset, err := h.svc.Update(c.Request.Context(), callerID, assetID, body.Title, body.Content)
	if writeAssetErr(c, err) {
		return
	}
	response.Success(c, asset)
}

func (h *Handler) Delete(c *gin.Context) {
	assetID := c.Param("id")
	callerID, ok := middleware.CallerID(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, response.ErrUnauthorized, "missing caller")
		return
	}

	err := h.svc.Delete(c.Request.Context(), callerID, assetID)
	if writeAssetErr(c, err) {
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) Share(c *gin.Context) {
	assetID := c.Param("id")
	callerID, ok := middleware.CallerID(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, response.ErrUnauthorized, "missing caller")
		return
	}

	var body struct {
		UserID      string `json:"user_id" binding:"required"`
		AccessLevel string `json:"access" binding:"required,oneof=read write"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		response.Error(c, http.StatusBadRequest, response.ErrBadRequest, err.Error())
		return
	}

	err := h.svc.Share(c.Request.Context(), callerID, assetID, body.UserID, body.AccessLevel)
	if writeAssetErr(c, err) {
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) RevokeShare(c *gin.Context) {
	assetID := c.Param("id")
	targetUserID := c.Param("userId")
	callerID, ok := middleware.CallerID(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, response.ErrUnauthorized, "missing caller")
		return
	}

	err := h.svc.RevokeShare(c.Request.Context(), callerID, assetID, targetUserID)
	if writeAssetErr(c, err) {
		return
	}
	c.Status(http.StatusNoContent)
}
