package router

import (
	"net/http"

	"github.com/dungpd/seta/core-service/internal/middleware"
	"github.com/dungpd/seta/core-service/internal/team"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

func New(jwks *middleware.JWKSClient, rdb *redis.Client, teamHandler *team.Handler) *gin.Engine {
	r := gin.Default()

	r.GET("/health", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	v1 := r.Group("/v1", middleware.JWTAuth(jwks, rdb))
	{
		v1.POST("/teams", teamHandler.CreateTeam)
		v1.POST("/teams/:id/members", teamHandler.AddMember)
		v1.DELETE("/teams/:id/members/:userId", teamHandler.RemoveMember)
	}

	return r
}
