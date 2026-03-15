package middleware

import (
	"net/http"
	"strings"

	"gateway/pkg/errors"

	"github.com/gin-gonic/gin"
)

const (
	// AuthHeaderKey 认证头键名
	AuthHeaderKey = "Authorization"
	// BearerPrefix Bearer前缀
	BearerPrefix = "Bearer "
	// APIKeyContextKey API Key在上下文中的键名
	APIKeyContextKey = "api_key"
	// UserIDContextKey 用户ID在上下文中的键名
	UserIDContextKey = "user_id"
)

// AuthConfig 认证配置
type AuthConfig struct {
	// Enabled 是否启用认证
	Enabled bool
	// ValidateFunc 自定义验证函数
	// 参数为API Key，返回用户ID和错误
	ValidateFunc func(apiKey string) (userID string, err error)
	// SkipPaths 跳过认证的路径
	SkipPaths []string
}

// Auth 认证中间件
func Auth(cfg *AuthConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 检查是否跳过认证
		if !cfg.Enabled || shouldSkipAuth(c.Request.URL.Path, cfg.SkipPaths) {
			c.Next()
			return
		}

		// 获取Authorization头
		authHeader := c.GetHeader(AuthHeaderKey)
		if authHeader == "" {
			abortWithError(c, errors.New(errors.ErrUnauthorized))
			return
		}

		// 提取API Key
		var apiKey string
		if strings.HasPrefix(authHeader, BearerPrefix) {
			apiKey = strings.TrimPrefix(authHeader, BearerPrefix)
		} else {
			apiKey = authHeader
		}

		if apiKey == "" {
			abortWithError(c, errors.NewWithDetails(errors.ErrUnauthorized, "missing api key"))
			return
		}

		// 验证API Key
		if cfg.ValidateFunc != nil {
			userID, err := cfg.ValidateFunc(apiKey)
			if err != nil {
				abortWithError(c, errors.NewWithDetails(errors.ErrUnauthorized, err.Error()))
				return
			}
			c.Set(UserIDContextKey, userID)
		}

		// 存储API Key到上下文
		c.Set(APIKeyContextKey, apiKey)
		c.Next()
	}
}

// shouldSkipAuth 检查是否应跳过认证
func shouldSkipAuth(path string, skipPaths []string) bool {
	for _, sp := range skipPaths {
		if strings.HasPrefix(path, sp) {
			return true
		}
	}
	return false
}

// abortWithError 中止请求并返回错误
func abortWithError(c *gin.Context, err *errors.APIError) {
	c.AbortWithStatusJSON(err.HTTPStatus(), gin.H{
		"error": gin.H{
			"code":    err.Code,
			"message": err.Message,
			"details": err.Details,
		},
	})
}

// GetAPIKey 从上下文获取API Key
func GetAPIKey(c *gin.Context) string {
	if v, exists := c.Get(APIKeyContextKey); exists {
		return v.(string)
	}
	return ""
}

// GetUserID 从上下文获取用户ID
func GetUserID(c *gin.Context) string {
	if v, exists := c.Get(UserIDContextKey); exists {
		return v.(string)
	}
	return ""
}

// SimpleAuth 简单认证中间件（仅检查API Key是否存在）
func SimpleAuth() gin.HandlerFunc {
	return Auth(&AuthConfig{
		Enabled: true,
		SkipPaths: []string{
			"/health",
			"/ready",
			"/metrics",
		},
	})
}

// NoAuth 无认证中间件（跳过所有认证）
func NoAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()
	}
}

// RequireAPIKey 要求API Key的中间件
func RequireAPIKey() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader(AuthHeaderKey)
		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": gin.H{
					"code":    errors.ErrUnauthorized,
					"message": "missing authorization header",
				},
			})
			return
		}

		var apiKey string
		if strings.HasPrefix(authHeader, BearerPrefix) {
			apiKey = strings.TrimPrefix(authHeader, BearerPrefix)
		} else {
			apiKey = authHeader
		}

		c.Set(APIKeyContextKey, apiKey)
		c.Next()
	}
}
