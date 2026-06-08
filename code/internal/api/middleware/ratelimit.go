package middleware

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

func RateLimiter(rdb *redis.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID := GetTenantID(c)
		if tenantID == "" {
			c.Next()
			return
		}

		maxRPS, _ := c.Get("tenant_max_rps")
		limit := 50
		if v, ok := maxRPS.(int); ok && v > 0 {
			limit = v
		}

		key := fmt.Sprintf("ratelimit:%s:%d", tenantID, time.Now().Unix())
		ctx := context.Background()

		count, err := rdb.Incr(ctx, key).Result()
		if err != nil {
			c.Next()
			return
		}
		if count == 1 {
			rdb.Expire(ctx, key, 2*time.Second)
		}

		c.Header("X-RateLimit-Limit", strconv.Itoa(limit))
		c.Header("X-RateLimit-Remaining", strconv.Itoa(max(0, limit-int(count))))

		if int(count) > limit {
			c.JSON(http.StatusTooManyRequests, gin.H{"error": gin.H{"code": "RATE_LIMITED", "message": "Request rate exceeded"}})
			c.Abort()
			return
		}
		c.Next()
	}
}
