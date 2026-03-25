package middleware

import (
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

type rateEntry struct {
	count       int
	windowStart time.Time
}

type IPRateLimiter struct {
	limitPerMinute int
	mu             sync.Mutex
	entries        map[string]rateEntry
}

func NewIPRateLimiter(limitPerMinute int) *IPRateLimiter {
	if limitPerMinute <= 0 {
		limitPerMinute = 240
	}
	return &IPRateLimiter{
		limitPerMinute: limitPerMinute,
		entries:        make(map[string]rateEntry),
	}
}

func (r *IPRateLimiter) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		ip := c.ClientIP()
		if ip == "" {
			ip = "unknown"
		}
		now := time.Now().UTC()

		r.mu.Lock()
		entry := r.entries[ip]
		if entry.windowStart.IsZero() || now.Sub(entry.windowStart) >= time.Minute {
			entry.windowStart = now
			entry.count = 0
		}
		entry.count++
		r.entries[ip] = entry

		if len(r.entries) > 2048 {
			r.evictOld(now)
		}
		r.mu.Unlock()

		if entry.count > r.limitPerMinute {
			retryAfter := int(time.Minute.Seconds()) - int(now.Sub(entry.windowStart).Seconds())
			if retryAfter < 1 {
				retryAfter = 1
			}
			c.Header("Retry-After", strconv.Itoa(retryAfter))
			AbortWithError(c, http.StatusTooManyRequests, "RATE_LIMITED", "too many requests")
			return
		}

		c.Next()
	}
}

func (r *IPRateLimiter) evictOld(now time.Time) {
	for ip, entry := range r.entries {
		if now.Sub(entry.windowStart) >= 2*time.Minute {
			delete(r.entries, ip)
		}
	}
}
