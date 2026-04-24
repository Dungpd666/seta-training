package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/dungpd/seta/auth-service/internal/auth"
	"github.com/dungpd/seta/auth-service/internal/response"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

const (
	CtxUserID = "user_id"
	CtxRole   = "role"
)

type TokenValidator interface {
	ParseToken(tokenStr string, opts ...jwt.ParserOption) (*auth.Claims, error)
	IsBlacklisted(ctx context.Context, jti string) (bool, error)
}

func JWTAuth(v TokenValidator) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if !strings.HasPrefix(authHeader, "Bearer ") {
			c.Abort()
			response.Error(c, http.StatusUnauthorized, response.ErrUnauthorized, "missing or invalid Authorization header")
			return
		}

		claims, err := v.ParseToken(
			strings.TrimPrefix(authHeader, "Bearer "),
			jwt.WithIssuer(auth.Issuer),
			jwt.WithAudience(auth.Audience),
			jwt.WithExpirationRequired(),
		)
		if err != nil {
			c.Abort()
			response.Error(c, http.StatusUnauthorized, response.ErrUnauthorized, "invalid token")
			return
		}

		blacklisted, err := v.IsBlacklisted(c.Request.Context(), claims.ID)
		if err != nil || blacklisted {
			c.Abort()
			response.Error(c, http.StatusUnauthorized, response.ErrUnauthorized, "token revoked")
			return
		}

		c.Set(CtxUserID, claims.Subject)
		c.Set(CtxRole, claims.Role)
		c.Next()
	}
}
