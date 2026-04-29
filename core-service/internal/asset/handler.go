package asset

import (
	"errors"
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

func (h *Handler) Create(c *gin.Context) {
	var body struct {
		ParentID *string `json:"parent_id"`
		Type     string  `json:"type" binding:"required"`
		Title    string  `json:"title" binding:"required"`
		Content  *string `json:"content"`
	}

	if err := c.ShouldBindJSON(&body); err != nil {
		response.Error(c, http.StatusBadRequest, response.ErrBadRequest, err.Error())
		return
	}

	ownerID, _ := c.Get("user_id")
	asset, err := h.svc.Create(c.Request.Context(), ownerID.(string), body.ParentID, body.Type, body.Title, body.Content)
	if errors.Is(err, ErrInvalidType) {
		response.Error(c, http.StatusBadRequest, response.ErrBadRequest, err.Error())
		return
	}
	if err != nil {
		response.Error(c, http.StatusInternalServerError, response.ErrInternal, err.Error())
		return
	}
	response.SuccessWithStatus(c, http.StatusCreated, asset)
}

func (h *Handler) GetByID(c *gin.Context) {
	assetID := c.Param("id")
	callerID, _ := c.Get("user_id")
	asset, err := h.svc.GetByID(c.Request.Context(), callerID.(string), assetID)
	if errors.Is(err, ErrNotFound) {
		response.Error(c, http.StatusNotFound, response.ErrNotFound, err.Error())
		return
	}
	if errors.Is(err, ErrForbidden) {
		response.Error(c, http.StatusForbidden, response.ErrForbidden, err.Error())
		return
	}
	if err != nil {
		response.Error(c, http.StatusInternalServerError, response.ErrInternal, err.Error())
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

	callerID, _ := c.Get("user_id")
	asset, err := h.svc.Update(c.Request.Context(), callerID.(string), assetID, body.Title, body.Content)
	if errors.Is(err, ErrNotFound) {
		response.Error(c, http.StatusNotFound, response.ErrNotFound, err.Error())
		return
	}
	if errors.Is(err, ErrForbidden) {
		response.Error(c, http.StatusForbidden, response.ErrForbidden, err.Error())
		return
	}
	if err != nil {
		response.Error(c, http.StatusInternalServerError, response.ErrInternal, err.Error())
		return
	}
	response.Success(c, asset)
}

func (h *Handler) Delete(c *gin.Context) {
	assetID := c.Param("id")
	callerID, _ := c.Get("user_id")

	err := h.svc.Delete(c.Request.Context(), callerID.(string), assetID)
	if errors.Is(err, ErrNotFound) {
		response.Error(c, http.StatusNotFound, response.ErrNotFound, err.Error())
		return
	}
	if errors.Is(err, ErrForbidden) {
		response.Error(c, http.StatusForbidden, response.ErrForbidden, err.Error())
		return
	}
	if err != nil {
		response.Error(c, http.StatusInternalServerError, response.ErrInternal, err.Error())
		return
	}
	c.Status(http.StatusNoContent)
}
