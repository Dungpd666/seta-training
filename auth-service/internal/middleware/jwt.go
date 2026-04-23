package middleware

import (
	"net/http"
	"strings"

	"github.com/dungpd/seta/auth-service/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

const (
	CtxUserID = "user_id"
	CtxRole   = "role"
)

type tokenValidator interface {
	ParseToken(tokenStr string, opts ...jwt.ParserOption) (*service.Claims, error)
	IsBlacklisted(jti string) (bool, error)
}

func JWTAuth(v tokenValidator) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if !strings.HasPrefix(authHeader, "Bearer ") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing or invalid Authorization header"})
			return
		}

		claims, err := v.ParseToken(
			strings.TrimPrefix(authHeader, "Bearer "),
			jwt.WithIssuer(service.Issuer),
			jwt.WithAudience(service.Audience),
			jwt.WithExpirationRequired(),
		)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
			return
		}

		blacklisted, err := v.IsBlacklisted(claims.ID)
		if err != nil || blacklisted {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "token revoked"})
			return
		}

		c.Set(CtxUserID, claims.Subject)
		c.Set(CtxRole, claims.Role)
		c.Next()
	}
}
