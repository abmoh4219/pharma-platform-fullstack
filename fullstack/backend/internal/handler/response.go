package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type apiError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func writeSuccess(c *gin.Context, status int, data any) {
	c.JSON(status, gin.H{
		"success": true,
		"data":    data,
	})
}

func writeError(c *gin.Context, status int, code, message string) {
	c.JSON(status, gin.H{
		"success": false,
		"error": apiError{
			Code:    code,
			Message: message,
		},
	})
}

func badRequest(c *gin.Context, code, message string) {
	writeError(c, http.StatusBadRequest, code, message)
}
