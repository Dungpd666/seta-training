package router

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func New() *gin.Engine {
	r := gin.Default()
	r.GET("/health", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})
	return r
}
