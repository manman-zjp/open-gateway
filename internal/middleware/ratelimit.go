package middleware

import (
	"net/http"
	"sync"
	"time"

	"gateway/pkg/errors"
	"github.com/gin-gonic/gin"
)

// RateLimiterConfig 限流配置
type RateLimiterConfig struct {
	// Enabled 是否启用
	Enabled bool
	// RequestsPerSecond 每秒请求数
	RequestsPerSecond int
	// RequestsPerMinute 每分钟请求数
	RequestsPerMinute int
	// RequestsPerHour 每小时请求数
	RequestsPerHour int
	// BurstSize 突发容量
	BurstSize int
	// KeyFunc 获取限流Key的函数
	KeyFunc func(*gin.Context) string
	// SkipPaths 跳过限流的路径
	SkipPaths []string
}

// DefaultRateLimiterConfig 默认限流配置
func DefaultRateLimiterConfig() *RateLimiterConfig {
	return &RateLimiterConfig{
		Enabled:           true,
		RequestsPerSecond: 10,
		RequestsPerMinute: 100,
		BurstSize:         20,
		KeyFunc: func(c *gin.Context) string {
			// 默认按IP限流
			return c.ClientIP()
		},
		SkipPaths: []string{"/health", "/ready", "/info"},
	}
}

// TokenBucket 令牌桶
type TokenBucket struct {
	tokens     float64
	capacity   float64
	refillRate float64 // 每秒补充的令牌数
	lastRefill time.Time
	mu         sync.Mutex
}

// NewTokenBucket 创建令牌桶
func NewTokenBucket(capacity float64, refillRate float64) *TokenBucket {
	return &TokenBucket{
		tokens:     capacity,
		capacity:   capacity,
		refillRate: refillRate,
		lastRefill: time.Now(),
	}
}

// Allow 是否允许请求
func (tb *TokenBucket) Allow() bool {
	tb.mu.Lock()
	defer tb.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(tb.lastRefill).Seconds()
	tb.lastRefill = now

	// 补充令牌
	tb.tokens += elapsed * tb.refillRate
	if tb.tokens > tb.capacity {
		tb.tokens = tb.capacity
	}

	// 消耗令牌
	if tb.tokens >= 1 {
		tb.tokens--
		return true
	}

	return false
}

// RateLimiter 限流器
type RateLimiter struct {
	config  *RateLimiterConfig
	buckets sync.Map // map[string]*TokenBucket
}

// NewRateLimiter 创建限流器
func NewRateLimiter(cfg *RateLimiterConfig) *RateLimiter {
	if cfg == nil {
		cfg = DefaultRateLimiterConfig()
	}
	return &RateLimiter{
		config: cfg,
	}
}

// getBucket 获取或创建令牌桶
func (rl *RateLimiter) getBucket(key string) *TokenBucket {
	if bucket, ok := rl.buckets.Load(key); ok {
		return bucket.(*TokenBucket)
	}

	capacity := float64(rl.config.BurstSize)
	refillRate := float64(rl.config.RequestsPerSecond)

	bucket := NewTokenBucket(capacity, refillRate)
	actual, _ := rl.buckets.LoadOrStore(key, bucket)
	return actual.(*TokenBucket)
}

// Allow 是否允许请求
func (rl *RateLimiter) Allow(key string) bool {
	bucket := rl.getBucket(key)
	return bucket.Allow()
}

// Cleanup 清理过期的桶（定期调用）
func (rl *RateLimiter) Cleanup() {
	// 简单实现：定期清理所有桶
	// 生产环境可以考虑LRU或TTL策略
}

// RateLimit 限流中间件
func RateLimit(cfg *RateLimiterConfig) gin.HandlerFunc {
	if cfg == nil {
		cfg = DefaultRateLimiterConfig()
	}

	limiter := NewRateLimiter(cfg)

	return func(c *gin.Context) {
		if !cfg.Enabled {
			c.Next()
			return
		}

		// 检查是否跳过
		for _, path := range cfg.SkipPaths {
			if c.Request.URL.Path == path {
				c.Next()
				return
			}
		}

		// 获取限流Key
		key := cfg.KeyFunc(c)
		if key == "" {
			key = c.ClientIP()
		}

		// 检查是否允许
		if !limiter.Allow(key) {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error": gin.H{
					"code":    errors.ErrTooManyRequests,
					"message": "rate limit exceeded",
				},
			})
			return
		}

		c.Next()
	}
}

// SlidingWindowLimiter 滑动窗口限流器
type SlidingWindowLimiter struct {
	windowSize  time.Duration
	maxRequests int
	requests    sync.Map // map[string][]time.Time
	mu          sync.Mutex
}

// NewSlidingWindowLimiter 创建滑动窗口限流器
func NewSlidingWindowLimiter(windowSize time.Duration, maxRequests int) *SlidingWindowLimiter {
	return &SlidingWindowLimiter{
		windowSize:  windowSize,
		maxRequests: maxRequests,
	}
}

// Allow 是否允许请求
func (swl *SlidingWindowLimiter) Allow(key string) bool {
	swl.mu.Lock()
	defer swl.mu.Unlock()

	now := time.Now()
	windowStart := now.Add(-swl.windowSize)

	// 获取或创建请求时间列表
	var times []time.Time
	if v, ok := swl.requests.Load(key); ok {
		times = v.([]time.Time)
	}

	// 移除窗口外的请求
	validTimes := make([]time.Time, 0, len(times))
	for _, t := range times {
		if t.After(windowStart) {
			validTimes = append(validTimes, t)
		}
	}

	// 检查是否超过限制
	if len(validTimes) >= swl.maxRequests {
		swl.requests.Store(key, validTimes)
		return false
	}

	// 记录本次请求
	validTimes = append(validTimes, now)
	swl.requests.Store(key, validTimes)

	return true
}

// SlidingWindowRateLimit 滑动窗口限流中间件
func SlidingWindowRateLimit(windowSize time.Duration, maxRequests int, keyFunc func(*gin.Context) string) gin.HandlerFunc {
	limiter := NewSlidingWindowLimiter(windowSize, maxRequests)

	return func(c *gin.Context) {
		key := keyFunc(c)
		if key == "" {
			key = c.ClientIP()
		}

		if !limiter.Allow(key) {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error": gin.H{
					"code":    errors.ErrTooManyRequests,
					"message": "rate limit exceeded",
				},
			})
			return
		}

		c.Next()
	}
}
