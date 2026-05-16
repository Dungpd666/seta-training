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
	limitStr := c.DefaultQuery("limit", "20")
	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 || limit > 100 {
		response.Error(c, http.StatusBadRequest, response.ErrBadRequest, "limit must be between 1 and 100")
		return
	}

	cursor := c.Query("cursor")

	users, total, err := h.svc.ListPage(c.Request.Context(), cursor, int32(limit))
	if WriteUserErr(c, err) {
		return
	}

	result := make([]UserResponse, len(users))
	for i, u := range users {
		result[i] = UserResponse{
			UserID:    u.UserID,
			Username:  u.Username,
			Email:     u.Email,
			Role:      u.Role,
			CreatedAt: u.CreatedAt,
		}
	}

	nextCursor := ""
	if len(users) == limit {
		nextCursor = users[len(users)-1].UserID
	}

	response.Paginated(c, result, response.PaginationMeta{
		Total:      total,
		Page:       0,
		Limit:      limit,
		NextCursor: nextCursor,
	})
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
