package response

import "github.com/gin-gonic/gin"

func Success(c *gin.Context, data any) {
	SuccessWithStatus(c, 200, data)
}

func SuccessWithStatus(c *gin.Context, status int, data any) {
	c.JSON(status, gin.H{"data": data})
}

func Error(c *gin.Context, status int, code, msg string) {
	reqID, _ := c.Get("request_id")
	reqIDStr, _ := reqID.(string)
	c.JSON(status, gin.H{"error": msg, "code": code, "request_id": reqIDStr})
}
