package middleware

import (
	"context"
	"sync"
	"time"

	"emergency-msystem-backend/pkg/response"

	"github.com/gin-gonic/gin"
)

// RateLimiter 令牌桶限流器
type RateLimiter struct {
	mu       sync.Mutex
	buckets  map[string]*tokenBucket
	rate     int           // 每分钟允许请求数
	capacity int
}

type tokenBucket struct {
	tokens    float64
	lastTime  time.Time
}

// NewRateLimiter 创建限流器
func NewRateLimiter(rpm int) *RateLimiter {
	return &RateLimiter{
		buckets:  make(map[string]*tokenBucket),
		rate:     rpm,
		capacity: rpm,
	}
}

// Allow 检查是否允许请求
func (rl *RateLimiter) Allow(key string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	bucket, exists := rl.buckets[key]
	now := time.Now()

	if !exists {
		rl.buckets[key] = &tokenBucket{
			tokens:   float64(rl.capacity - 1),
			lastTime: now,
		}
		return true
	}

	// 补充令牌
	elapsed := now.Sub(bucket.lastTime).Minutes()
	bucket.tokens += elapsed * float64(rl.rate)
	if bucket.tokens > float64(rl.capacity) {
		bucket.tokens = float64(rl.capacity)
	}
	bucket.lastTime = now

	if bucket.tokens >= 1 {
		bucket.tokens--
		return true
	}

	return false
}

// Cleanup 定期清理过期桶
func (rl *RateLimiter) Cleanup(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			rl.mu.Lock()
			threshold := time.Now().Add(-5 * time.Minute)
			for key, bucket := range rl.buckets {
				if bucket.lastTime.Before(threshold) {
					delete(rl.buckets, key)
				}
			}
			rl.mu.Unlock()
		}
	}
}

// RateLimit 限流中间件
func RateLimit(rl *RateLimiter) gin.HandlerFunc {
	return func(c *gin.Context) {
		key := c.ClientIP()
		if userID, exists := c.Get("user_id"); exists {
			key = userID.(string)
		}

		if !rl.Allow(key) {
			response.Error(c, 429, 42900, "too many requests")
			c.Abort()
			return
		}

		c.Next()
	}
}
