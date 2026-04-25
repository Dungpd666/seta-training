package router

import (
	"context"

	"github.com/dungpd/seta/auth-service/internal/auth"
	"github.com/dungpd/seta/auth-service/internal/middleware"
	"github.com/dungpd/seta/auth-service/internal/user"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog/log"
)

func New(
	dbPool *pgxpool.Pool,
	authHandler *auth.Handler,
	userHandler *user.Handler,
	v middleware.TokenValidator,
) *gin.Engine {
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(requestLogger())

	r.GET("/health", healthCheck(dbPool))
	r.GET("/.well-known/jwks.json", authHandler.JWKS)

	v1 := r.Group("/v1")
	registerAuthRoutes(v1, authHandler)
	registerUserRoutes(v1, userHandler, v)

	return r
}

func requestLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		log.Info().
			Str("method", c.Request.Method).
			Str("path", c.Request.URL.Path).
			Msg("request")
		c.Next()
	}
}

func healthCheck(dbPool *pgxpool.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		if err := dbPool.Ping(context.Background()); err != nil {
			c.JSON(503, gin.H{"status": "unhealthy"})
			return
		}
		c.JSON(200, gin.H{"status": "ok"})
	}
}

func registerAuthRoutes(rg *gin.RouterGroup, h *auth.Handler) {
	rg.POST("/register", h.Register)
	rg.POST("/login", h.Login)
	rg.POST("/refresh", h.Refresh)
	rg.POST("/logout", h.Logout)
}

func registerUserRoutes(rg *gin.RouterGroup, h *user.Handler, v middleware.TokenValidator) {
	protected := rg.Group("/")
	protected.Use(middleware.JWTAuth(v))
	protected.GET("/users", h.ListUsers)
	protected.POST("/users/import", h.ImportUsers)
}
