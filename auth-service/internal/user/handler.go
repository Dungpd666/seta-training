package user

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/dungpd/seta/auth-service/internal/response"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
)

const maxImportFileSize = 10 << 20 // 10MB

func WriteUserErr(c *gin.Context, err error) bool {
	switch {
	case errors.Is(err, ErrEmailInUse):
		response.Error(c, http.StatusConflict, response.ErrConflict, err.Error())
	case errors.Is(err, ErrInvalidCredentials):
		response.Error(c, http.StatusUnauthorized, response.ErrUnauthorized, err.Error())
	case err != nil:
		log.Error().Err(err).Msg("internal error")
		response.Error(c, http.StatusInternalServerError, response.ErrInternal, "internal server error")
	default:
		return false
	}
	return true
}

type Handler struct {
	svc Service
}

func NewHandler(svc Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) ListUsers(c *gin.Context) {
	users, err := h.svc.ListAll(c.Request.Context())
	if WriteUserErr(c, err) {
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

func (h *Handler) ImportUsers(c *gin.Context) {
	c.Request.Body = http.MaxBytesReader(
		c.Writer,
		c.Request.Body,
		maxImportFileSize,
	)

	fileHeader, err := c.FormFile("file")
	if err != nil {
		response.Error(c, http.StatusBadRequest, response.ErrBadRequest, "missing 'file' field: "+err.Error())
		return
	}

	if fileHeader.Size > maxImportFileSize {
		response.Error(c, http.StatusRequestEntityTooLarge, response.ErrBadRequest, "file size exceeds limit(10MB)")
		return
	}

	file, err := fileHeader.Open()
	if err != nil {
		response.Error(c, http.StatusInternalServerError, response.ErrInternal, "failed to open file")
		return
	}

	defer file.Close()

	workers, _ := strconv.Atoi(c.Query("workers")) // 0 if missing/invalid → service uses default

	result, err := h.svc.ImportFromCSV(c.Request.Context(), file, workers)
	if err != nil {
		response.Error(c, http.StatusBadRequest, response.ErrBadRequest, "failed to import users: "+err.Error())
		return
	}

	response.Success(c, result)
}
