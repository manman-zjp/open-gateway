package middleware

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"gateway/apikey"
	"gateway/model"
)

// APIKeyAuth API Key 认证中间件
func APIKeyAuth(service *apikey.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 从 Header 获取 Authorization
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			// 尝试从查询参数获取
			authHeader = c.Query("api_key")
		}

		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": gin.H{
					"message": "API key is required",
					"type":    "authentication_error",
					"code":    "missing_api_key",
				},
			})
			c.Abort()
			return
		}

		// 提取 API Key
		rawKey := apikey.ExtractAPIKey(authHeader)
		if rawKey == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": gin.H{
					"message": "Invalid API key format",
					"type":    "authentication_error",
					"code":    "invalid_api_key",
				},
			})
			c.Abort()
			return
		}

		// 验证 API Key
		key, err := service.Validate(c.Request.Context(), rawKey)
		if err != nil {
			RecordAPIKeyRequest("unknown", "invalid")
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": gin.H{
					"message": "Invalid or expired API key",
					"type":    "authentication_error",
					"code":    "invalid_api_key",
				},
			})
			c.Abort()
			return
		}

		// 将 API Key 信息存入 context
		c.Set("api_key", key)
		c.Set("api_key_id", key.ID)
		c.Set("user_id", key.UserID)

		// 记录指标
		RecordAPIKeyRequest(key.ID, "success")

		c.Next()
	}
}

// APIKeyOptional 可选的 API Key 认证
// 如果提供了 API Key 则验证，否则继续
func APIKeyOptional(service *apikey.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			authHeader = c.Query("api_key")
		}

		if authHeader != "" {
			rawKey := apikey.ExtractAPIKey(authHeader)
			if rawKey != "" {
				key, err := service.Validate(c.Request.Context(), rawKey)
				if err == nil {
					c.Set("api_key", key)
					c.Set("api_key_id", key.ID)
					c.Set("user_id", key.UserID)
				}
			}
		}

		c.Next()
	}
}

// RequireModel 检查是否有权限访问指定模型
func RequireModel() gin.HandlerFunc {
	return func(c *gin.Context) {
		key, exists := c.Get("api_key")
		if !exists {
			c.Next()
			return
		}

		apiKey := key.(*model.APIKey)

		// 从请求体中获取模型名称（需要先读取然后恢复）
		var modelName string
		if c.Request.Body != nil {
			bodyBytes, err := io.ReadAll(c.Request.Body)
			if err == nil {
				// 恢复 body
				c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

				// 简单解析获取 model 字段
				var req struct {
					Model string `json:"model"`
				}
				if json.Unmarshal(bodyBytes, &req) == nil {
					modelName = req.Model
				}
			}
		}

		if modelName != "" {
			// 存入 context 供后续使用
			c.Set("request_model", modelName)

			if !apiKey.CanAccessModel(modelName) {
				c.JSON(http.StatusForbidden, gin.H{
					"error": gin.H{
						"message": "API key does not have access to model: " + modelName,
						"type":    "authorization_error",
						"code":    "model_access_denied",
					},
				})
				c.Abort()
				return
			}
		}

		c.Next()
	}
}

// RequireProvider 检查是否有权限访问指定提供商
func RequireProvider(providerName string) gin.HandlerFunc {
	return func(c *gin.Context) {
		key, exists := c.Get("api_key")
		if !exists {
			c.Next()
			return
		}

		apiKey := key.(*model.APIKey)
		if !apiKey.CanAccessProvider(providerName) {
			c.JSON(http.StatusForbidden, gin.H{
				"error": gin.H{
					"message": "API key does not have access to provider: " + providerName,
					"type":    "authorization_error",
					"code":    "provider_access_denied",
				},
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// APIKeyRateLimit API Key 级别的限流
func APIKeyRateLimit(service *apikey.Service) gin.HandlerFunc {
	limiters := make(map[string]*TokenBucket)
	var mu = make(chan struct{}, 1)
	mu <- struct{}{}

	return func(c *gin.Context) {
		key, exists := c.Get("api_key")
		if !exists {
			c.Next()
			return
		}

		apiKey := key.(*model.APIKey)
		if apiKey.RateLimit <= 0 {
			c.Next()
			return
		}

		// 获取或创建限流器
		<-mu
		limiter, ok := limiters[apiKey.ID]
		if !ok {
			limiter = NewTokenBucket(float64(apiKey.RateLimit), float64(apiKey.RateLimit))
			limiters[apiKey.ID] = limiter
		}
		mu <- struct{}{}

		if !limiter.Allow() {
			RecordRateLimitHit("api_key")
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error": gin.H{
					"message": "Rate limit exceeded for this API key",
					"type":    "rate_limit_error",
					"code":    "rate_limit_exceeded",
				},
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// GetAPIKeyModel 从 context 获取 API Key 模型
func GetAPIKeyModel(c *gin.Context) *model.APIKey {
	if key, exists := c.Get("api_key"); exists {
		return key.(*model.APIKey)
	}
	return nil
}

// GetAPIKeyIDFromContext 从 context 获取 API Key ID
func GetAPIKeyIDFromContext(c *gin.Context) string {
	if id, exists := c.Get("api_key_id"); exists {
		return id.(string)
	}
	return ""
}

// GetUserIDFromContext 从 context 获取 User ID
func GetUserIDFromContext(c *gin.Context) string {
	if id, exists := c.Get("user_id"); exists {
		if str, ok := id.(string); ok {
			return str
		}
	}
	return ""
}

// AdminAuth 管理员认证中间件
func AdminAuth(adminKey string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if adminKey == "" {
			c.Next()
			return
		}

		authHeader := c.GetHeader("X-Admin-Key")
		if authHeader == "" {
			authHeader = c.GetHeader("Authorization")
			if strings.HasPrefix(authHeader, "Bearer ") {
				authHeader = strings.TrimPrefix(authHeader, "Bearer ")
			}
		}

		if authHeader != adminKey {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": gin.H{
					"message": "Invalid admin key",
					"type":    "authentication_error",
					"code":    "invalid_admin_key",
				},
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// CheckTokenQuota 检查 Token 额度
func CheckTokenQuota() gin.HandlerFunc {
	return func(c *gin.Context) {
		key, exists := c.Get("api_key")
		if !exists {
			c.Next()
			return
		}

		apiKey := key.(*model.APIKey)

		// 检查总 Token 额度
		if apiKey.TokenQuota > 0 && apiKey.UsedTokens >= apiKey.TokenQuota {
			c.JSON(http.StatusPaymentRequired, gin.H{
				"error": gin.H{
					"message": "Token quota exceeded. Used: " + formatInt64(apiKey.UsedTokens) + ", Quota: " + formatInt64(apiKey.TokenQuota),
					"type":    "quota_exceeded_error",
					"code":    "token_quota_exceeded",
				},
			})
			c.Abort()
			return
		}

		// 检查每日 Token 额度
		if apiKey.DailyTokenQuota > 0 && apiKey.UsedDailyTokens >= apiKey.DailyTokenQuota {
			c.JSON(http.StatusPaymentRequired, gin.H{
				"error": gin.H{
					"message": "Daily token quota exceeded. Used: " + formatInt64(apiKey.UsedDailyTokens) + ", Quota: " + formatInt64(apiKey.DailyTokenQuota),
					"type":    "quota_exceeded_error",
					"code":    "daily_token_quota_exceeded",
				},
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// GetRequestModel 从 context 获取请求的模型名称
func GetRequestModel(c *gin.Context) string {
	if m, exists := c.Get("request_model"); exists {
		return m.(string)
	}
	return ""
}

func formatInt64(n int64) string {
	if n >= 1000000 {
		return fmt.Sprintf("%.1fM", float64(n)/1000000)
	}
	if n >= 1000 {
		return fmt.Sprintf("%.1fK", float64(n)/1000)
	}
	return fmt.Sprintf("%d", n)
}
