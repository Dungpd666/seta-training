package handler

import (
	"crypto/rsa"
	"net/http"
	"strings"

	"github.com/dungpd/seta/auth-service/internal/domain"
	"github.com/gin-gonic/gin"
)

type userService interface {
	Register(username, email, password, role string) (*domain.User, error)
	Login(email, password string) (*domain.User, error)
	ListAll() ([]domain.User, error)
}

type authService interface {
	GenerateTokenPair(userID, role string) (string, string, error)
	RotateRefreshToken(tokenStr string) (string, string, error)
	RevokeSession(accessTokenStr, refreshTokenStr string) error
	PublicKey() *rsa.PublicKey
}

type UserHandler struct {
	userSvc userService
	authSvc authService
}

func NewUserHandler(userSvc userService, authSvc authService) *UserHandler {
	return &UserHandler{userSvc: userSvc, authSvc: authSvc}
}

type registerRequest struct {
	Username string `json:"username" binding:"required"`
	Email    string `json:"email"    binding:"required,email"`
	Password string `json:"password" binding:"required,min=6"`
	Role     string `json:"role"     binding:"required,oneof=manager member"`
}

func (h *UserHandler) Register(c *gin.Context) {
	var req registerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	user, err := h.userSvc.Register(req.Username, req.Email, req.Password, req.Role)
	if err != nil {
		status := http.StatusBadRequest
		if err.Error() == "email already in use" {
			status = http.StatusConflict
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{
		"user_id":  user.UserID,
		"username": user.Username,
		"email":    user.Email,
		"role":     user.Role,
	})
}

type loginRequest struct {
	Email    string `json:"email"    binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

func (h *UserHandler) Login(c *gin.Context) {
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	user, err := h.userSvc.Login(req.Email, req.Password)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
		return
	}
	accessToken, refreshToken, err := h.authSvc.GenerateTokenPair(user.UserID, user.Role)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate tokens"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"access_token": accessToken, "refresh_token": refreshToken})
}

func (h *UserHandler) Refresh(c *gin.Context) {
	var req struct {
		RefreshToken string `json:"refresh_token" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	accessToken, refreshToken, err := h.authSvc.RotateRefreshToken(req.RefreshToken)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"access_token": accessToken, "refresh_token": refreshToken})
}

func (h *UserHandler) Logout(c *gin.Context) {
	authHeader := c.GetHeader("Authorization")
	if !strings.HasPrefix(authHeader, "Bearer ") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing authorization header"})
		return
	}
	var req struct {
		RefreshToken string `json:"refresh_token" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := h.authSvc.RevokeSession(strings.TrimPrefix(authHeader, "Bearer "), req.RefreshToken); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *UserHandler) ListUsers(c *gin.Context) {
	users, err := h.userSvc.ListAll()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list users"})
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
	c.JSON(http.StatusOK, result)
}
