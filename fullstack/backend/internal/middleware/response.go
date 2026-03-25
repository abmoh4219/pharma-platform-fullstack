package middleware

import "github.com/gin-gonic/gin"

func AbortWithError(c *gin.Context, status int, code, message string) {
	c.AbortWithStatusJSON(status, gin.H{
		"success": false,
		"error":   message,
		"code":    code,
	})
}
