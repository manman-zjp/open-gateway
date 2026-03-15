package router

import (
	"gateway/apikey"
	"gateway/internal/handler"
	"gateway/internal/middleware"
	"gateway/web"

	"github.com/gin-gonic/gin"
)

// Config 路由配置
type Config struct {
	Mode                  string
	EnableAuth            bool
	AuthSkipPaths         []string
	EnableRateLimit       bool
	RateLimitRPS          int
	EnableMetrics         bool
	EnableWebUI           bool
	AdminKey              string
	APIKeyService         *apikey.Service
	ProviderConfigHandler *handler.ProviderConfigHandler
}

// DefaultConfig 默认配置
func DefaultConfig() *Config {
	return &Config{
		Mode:            gin.ReleaseMode,
		EnableAuth:      false,
		EnableRateLimit: false,
		RateLimitRPS:    10,
		EnableMetrics:   true,
		EnableWebUI:     true,
		AuthSkipPaths: []string{
			"/health",
			"/ready",
			"/info",
			"/metrics",
		},
	}
}

// New 创建路由引擎
func New(cfg *Config) *gin.Engine {
	if cfg == nil {
		cfg = DefaultConfig()
	}

	gin.SetMode(cfg.Mode)
	r := gin.New()

	// 全局中间件
	r.Use(middleware.Recovery())
	r.Use(middleware.RequestID())
	r.Use(middleware.Logger())
	r.Use(middleware.CORS(nil))

	// Prometheus 指标
	if cfg.EnableMetrics {
		r.Use(middleware.Prometheus())
		r.GET("/metrics", middleware.PrometheusHandler())
	}

	// 限流中间件
	if cfg.EnableRateLimit {
		r.Use(middleware.RateLimit(&middleware.RateLimiterConfig{
			Enabled:           true,
			RequestsPerSecond: cfg.RateLimitRPS,
			BurstSize:         cfg.RateLimitRPS * 2,
			SkipPaths:         []string{"/health", "/ready", "/info"},
		}))
	}

	// 健康检查路由
	healthHandler := handler.NewHealthHandler()
	r.GET("/health", healthHandler.Health)
	r.GET("/ready", healthHandler.Ready)
	r.GET("/info", healthHandler.Info)

	// API路由组
	api := r.Group("/v1")
	{
		if cfg.EnableAuth {
			api.Use(middleware.Auth(&middleware.AuthConfig{
				Enabled:   true,
				SkipPaths: cfg.AuthSkipPaths,
			}))
		}

		proxyHandler := handler.NewProxyHandler()

		// Chat Completions
		api.POST("/chat/completions", proxyHandler.ChatCompletion)

		// Models
		api.GET("/models", proxyHandler.ListModels)
		api.GET("/models/:model", proxyHandler.GetModel)

		// Images
		api.POST("/images/generations", proxyHandler.ImageGeneration)

		// Audio
		api.POST("/audio/transcriptions", proxyHandler.AudioTranscription)
		api.POST("/audio/speech", proxyHandler.AudioSpeech)

		// Embeddings
		api.POST("/embeddings", proxyHandler.Embeddings)

		// Stats (内部接口)
		api.GET("/stats", proxyHandler.GetStats)
	}

	// API Key 管理路由
	if cfg.APIKeyService != nil {
		apiKeyHandler := handler.NewAPIKeyHandler(cfg.APIKeyService)
		handler.RegisterAPIKeyRoutes(&r.RouterGroup, apiKeyHandler, cfg.AdminKey)
	}

	// Provider 配置管理路由
	if cfg.ProviderConfigHandler != nil {
		handler.RegisterProviderConfigRoutes(&r.RouterGroup, cfg.ProviderConfigHandler, cfg.AdminKey)
	}

	// Web UI
	if cfg.EnableWebUI {
		web.RegisterRoutes(r)
	}

	return r
}
