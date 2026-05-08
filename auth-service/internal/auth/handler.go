package auth

import (
	"encoding/base64"
	"encoding/binary"
	"net/http"
	"strings"

	"github.com/dungpd/seta/auth-service/internal/response"
	"github.com/dungpd/seta/auth-service/internal/user"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
)

func writeAuthErr(c *gin.Context, err error) bool {
	switch {
	case err != nil:
		response.Error(c, http.StatusUnauthorized, response.ErrUnauthorized, err.Error())
	default:
		return false
	}
	return true
}

type RegisterRequest struct {
	Username string `json:"username" binding:"required"`
	Email    string `json:"email"    binding:"required,email"`
	Password string `json:"password" binding:"required,min=6"`
	Role     string `json:"role"     binding:"required,oneof=manager member"`
}

type LoginRequest struct {
	Email    string `json:"email"    binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

type RefreshRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

type LogoutRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

type Handler struct {
	userSvc user.Service
	authSvc Service
}

func NewHandler(userSvc user.Service, authSvc Service) *Handler {
	return &Handler{userSvc: userSvc, authSvc: authSvc}
}

func (h *Handler) Register(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, response.ErrBadRequest, err.Error())
		return
	}
	u, err := h.userSvc.Register(c.Request.Context(), req.Username, req.Email, req.Password, req.Role)
	if user.WriteUserErr(c, err) {
		return
	}
	response.SuccessWithStatus(c, http.StatusCreated, gin.H{
		"user_id":  u.UserID,
		"username": u.Username,
		"email":    u.Email,
		"role":     u.Role,
	})
}

func (h *Handler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, response.ErrBadRequest, err.Error())
		return
	}
	ctx := c.Request.Context()
	u, err := h.userSvc.Login(ctx, req.Email, req.Password)
	if user.WriteUserErr(c, err) {
		return
	}
	accessToken, refreshToken, err := h.authSvc.GenerateTokenPair(ctx, u.UserID, u.Role)
	if err != nil {
		log.Error().Err(err).Msg("internal error")
		response.Error(c, http.StatusInternalServerError, response.ErrInternal, "failed to generate tokens")
		return
	}
	response.Success(c, gin.H{"access_token": accessToken, "refresh_token": refreshToken})
}

func (h *Handler) Refresh(c *gin.Context) {
	var req RefreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, response.ErrBadRequest, err.Error())
		return
	}
	accessToken, refreshToken, err := h.authSvc.RotateRefreshToken(c.Request.Context(), req.RefreshToken)
	if writeAuthErr(c, err) {
		return
	}
	response.Success(c, gin.H{"access_token": accessToken, "refresh_token": refreshToken})
}

func (h *Handler) Logout(c *gin.Context) {
	authHeader := c.GetHeader("Authorization")
	if !strings.HasPrefix(authHeader, "Bearer ") {
		response.Error(c, http.StatusBadRequest, response.ErrBadRequest, "missing authorization header")
		return
	}
	var req LogoutRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, response.ErrBadRequest, err.Error())
		return
	}
	if err := h.authSvc.RevokeSession(c.Request.Context(), strings.TrimPrefix(authHeader, "Bearer "), req.RefreshToken); err != nil {
		response.Error(c, http.StatusUnauthorized, response.ErrUnauthorized, err.Error())
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) JWKS(c *gin.Context) {
	pub := h.authSvc.PublicKey()
	n := base64.RawURLEncoding.EncodeToString(pub.N.Bytes())

	eBuf := make([]byte, 4)
	binary.BigEndian.PutUint32(eBuf, uint32(pub.E))
	i := 0
	for i < len(eBuf)-1 && eBuf[i] == 0 {
		i++
	}
	e := base64.RawURLEncoding.EncodeToString(eBuf[i:])

	response.Success(c, gin.H{"keys": []gin.H{{
		"kty": "RSA", "use": "sig", "alg": "RS256",
		"kid": "auth-service-key-1", "n": n, "e": e,
	}}})
}
