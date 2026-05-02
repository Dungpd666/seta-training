package middleware

import "github.com/gin-gonic/gin"

func CallerID(c *gin.Context) (string, bool) {
	v, ok := c.Get(CtxUserID)
	if !ok {
		return "", false
	}
	id, ok := v.(string)
	return id, ok
}

func CallerRole(c *gin.Context) (string, bool) {
	v, ok := c.Get(CtxRole)
	if !ok {
		return "", false
	}
	role, ok := v.(string)
	return role, ok
}
