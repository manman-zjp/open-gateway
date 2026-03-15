package handler

import (
	"net/http"

	"gateway/cache"
	"gateway/global"
	"gateway/provider"
	"gateway/storage"

	"github.com/gin-gonic/gin"
)

// HealthHandler 健康检查处理器
type HealthHandler struct{}

// NewHealthHandler 创建健康检查处理器
func NewHealthHandler() *HealthHandler {
	return &HealthHandler{}
}

// Health 存活探针
// GET /health
func (h *HealthHandler) Health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":  "healthy",
		"version": global.Version,
	})
}

// Ready 就绪探针
// GET /ready
func (h *HealthHandler) Ready(c *gin.Context) {
	ctx := c.Request.Context()
	checks := make(map[string]string)
	allReady := true

	// 检查存储
	if s := storage.GetStorage(); s != nil {
		if err := s.HealthCheck(ctx); err != nil {
			checks["storage"] = "unhealthy: " + err.Error()
			allReady = false
		} else {
			checks["storage"] = "healthy"
		}
	} else {
		checks["storage"] = "not configured"
	}

	// 检查缓存
	if ca := cache.GetCache(); ca != nil {
		if err := ca.HealthCheck(ctx); err != nil {
			checks["cache"] = "unhealthy: " + err.Error()
			allReady = false
		} else {
			checks["cache"] = "healthy"
		}
	} else {
		checks["cache"] = "not configured"
	}

	// 检查提供商
	providers := provider.GetAllProviders()
	providerChecks := make(map[string]string)
	for name, p := range providers {
		if err := p.HealthCheck(ctx); err != nil {
			providerChecks[name] = "unhealthy: " + err.Error()
			// 提供商不健康不影响整体就绪状态
		} else {
			providerChecks[name] = "healthy"
		}
	}
	checks["providers"] = "checked"

	status := http.StatusOK
	statusText := "ready"
	if !allReady {
		status = http.StatusServiceUnavailable
		statusText = "not ready"
	}

	c.JSON(status, gin.H{
		"status":    statusText,
		"checks":    checks,
		"providers": providerChecks,
		"version":   global.Version,
	})
}

// Info 获取服务信息
// GET /info
func (h *HealthHandler) Info(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"name":       "AI Gateway",
		"version":    global.Version,
		"build_time": global.BuildTime,
		"git_commit": global.GitCommit,
		"providers":  provider.ListProviders(),
		"factories":  provider.ListFactories(),
	})
}
