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
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(middleware.RequestID())

	r.GET("/health", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	v1 := r.Group("/v1", middleware.JWTAuth(jwks, rdb))
	{
		v1.GET("/assets/:id", assetHandler.GetByID)
		v1.GET("/teams/:id/members", teamHandler.GetMembers)
		v1.POST("/assets", assetHandler.Create)
		v1.POST("/teams", teamHandler.CreateTeam)
		v1.POST("/teams/:id/members", teamHandler.AddMember)
		v1.POST("/teams/:id/managers", teamHandler.AddManager)
		v1.POST("/assets/:id/share", assetHandler.Share)
		v1.PUT("/assets/:id", assetHandler.Update)
		v1.DELETE("/assets/:id", assetHandler.Delete)
		v1.DELETE("/teams/:id/members/:userId", teamHandler.RemoveMember)
		v1.DELETE("/teams/:id/managers/:userId", teamHandler.RemoveManager)
		v1.DELETE("/assets/:id/share/:userId", assetHandler.RevokeShare)
	}

	return r
}
