package middleware

import (
	"crypto/rand"
	"encoding/hex"
	"log"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

const ContextRequestID = "request_id"

func RequestContext() gin.HandlerFunc {
	return func(c *gin.Context) {
		reqID := strings.TrimSpace(c.GetHeader("X-Request-ID"))
		if reqID == "" {
			reqID = newRequestID()
		}
		c.Set(ContextRequestID, reqID)
		c.Writer.Header().Set("X-Request-ID", reqID)

		start := time.Now()
		c.Next()

		latency := time.Since(start)
		log.Printf(
			`request_id=%s method=%s path=%s status=%d latency_ms=%d ip=%s`,
			reqID,
			c.Request.Method,
			c.Request.URL.Path,
			c.Writer.Status(),
			latency.Milliseconds(),
			c.ClientIP(),
		)
	}
}

func newRequestID() string {
	buf := make([]byte, 8)
	if _, err := rand.Read(buf); err != nil {
		return "req_fallback"
	}
	return "req_" + hex.EncodeToString(buf)
}
