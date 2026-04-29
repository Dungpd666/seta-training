package router

import (
	"net/http"

	"github.com/dungpd/seta/core-service/internal/asset"
	"github.com/dungpd/seta/core-service/internal/middleware"
	"github.com/dungpd/seta/core-service/internal/team"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

func New(jwks *middleware.JWKSClient, rdb *redis.Client, teamHandler *team.Handler, assetHandler *asset.Handler) *gin.Engine {
	r := gin.Default()

	r.GET("/health", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	v1 := r.Group("/v1", middleware.JWTAuth(jwks, rdb))
	{
		v1.GET("/assets/:id", assetHandler.GetByID)
		v1.POST("/assets", assetHandler.Create)
		v1.POST("/teams", teamHandler.CreateTeam)
		v1.POST("/teams/:id/members", teamHandler.AddMember)
		v1.POST("/teams/:id/managers", teamHandler.AddManager)
		v1.PUT("/assets/:id", assetHandler.Update)
		v1.DELETE("/assets/:id", assetHandler.Delete)
		v1.DELETE("/teams/:id/members/:userId", teamHandler.RemoveMember)
		v1.DELETE("/teams/:id/managers/:userId", teamHandler.RemoveManager)
	}

	return r
}
