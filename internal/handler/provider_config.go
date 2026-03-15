package handler

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"

	"gateway/model"
	"gateway/provider"
)

// ProviderConfigHandler Provider 配置管理处理器
type ProviderConfigHandler struct {
	configs map[string]*model.ProviderConfig
	mu      sync.RWMutex
}

// NewProviderConfigHandler 创建处理器
func NewProviderConfigHandler() *ProviderConfigHandler {
	return &ProviderConfigHandler{
		configs: make(map[string]*model.ProviderConfig),
	}
}

// List 列出所有 Provider 配置
// GET /admin/providers
func (h *ProviderConfigHandler) List(c *gin.Context) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	var configs []*model.ProviderConfig
	for _, cfg := range h.configs {
		// 不返回 API Key 明文
		cfgCopy := *cfg
		if cfgCopy.APIKey != "" {
			cfgCopy.APIKey = maskAPIKey(cfgCopy.APIKey)
		}
		configs = append(configs, &cfgCopy)
	}

	c.JSON(http.StatusOK, model.ProviderConfigListResponse{
		Total: len(configs),
		Data:  configs,
	})
}

// Get 获取单个 Provider 配置
// GET /admin/providers/:id
func (h *ProviderConfigHandler) Get(c *gin.Context) {
	id := c.Param("id")

	h.mu.RLock()
	cfg, exists := h.configs[id]
	h.mu.RUnlock()

	if !exists {
		c.JSON(http.StatusNotFound, gin.H{
			"error": gin.H{"message": "Provider not found", "type": "not_found_error"},
		})
		return
	}

	cfgCopy := *cfg
	if cfgCopy.APIKey != "" {
		cfgCopy.APIKey = maskAPIKey(cfgCopy.APIKey)
	}
	c.JSON(http.StatusOK, cfgCopy)
}

// Create 创建 Provider 配置
// POST /admin/providers
func (h *ProviderConfigHandler) Create(c *gin.Context) {
	var req model.ProviderConfigCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{"message": "Invalid request: " + err.Error(), "type": "invalid_request_error"},
		})
		return
	}

	// 检查 Provider 类型是否支持
	if _, exists := provider.GetFactory(req.Type); !exists {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"message": "Unsupported provider type: " + req.Type,
				"type":    "invalid_request_error",
			},
		})
		return
	}

	now := time.Now()
	cfg := &model.ProviderConfig{
		ID:        generateID(),
		Name:      req.Name,
		Type:      req.Type,
		BaseURL:   req.BaseURL,
		APIKey:    req.APIKey,
		Status:    model.ProviderStatusActive,
		Timeout:   req.Timeout,
		Models:    req.Models,
		Extra:     req.Extra,
		CreatedAt: now,
		UpdatedAt: now,
	}

	if cfg.Timeout <= 0 {
		cfg.Timeout = 60
	}

	// 尝试初始化 Provider
	if err := h.initProvider(cfg); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"message": "Failed to initialize provider: " + err.Error(),
				"type":    "invalid_request_error",
			},
		})
		return
	}

	h.mu.Lock()
	h.configs[cfg.ID] = cfg
	h.mu.Unlock()

	// 返回时隐藏 API Key
	cfgCopy := *cfg
	if cfgCopy.APIKey != "" {
		cfgCopy.APIKey = maskAPIKey(cfgCopy.APIKey)
	}
	c.JSON(http.StatusCreated, cfgCopy)
}

// Update 更新 Provider 配置
// PATCH /admin/providers/:id
func (h *ProviderConfigHandler) Update(c *gin.Context) {
	id := c.Param("id")

	var req model.ProviderConfigUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{"message": "Invalid request: " + err.Error(), "type": "invalid_request_error"},
		})
		return
	}

	h.mu.Lock()
	cfg, exists := h.configs[id]
	if !exists {
		h.mu.Unlock()
		c.JSON(http.StatusNotFound, gin.H{
			"error": gin.H{"message": "Provider not found", "type": "not_found_error"},
		})
		return
	}

	// 更新字段
	if req.Name != nil {
		cfg.Name = *req.Name
	}
	if req.BaseURL != nil {
		cfg.BaseURL = *req.BaseURL
	}
	if req.APIKey != nil {
		cfg.APIKey = *req.APIKey
	}
	if req.Status != nil {
		cfg.Status = *req.Status
	}
	if req.Timeout != nil {
		cfg.Timeout = *req.Timeout
	}
	if req.Models != nil {
		cfg.Models = req.Models
	}
	if req.Extra != nil {
		cfg.Extra = req.Extra
	}
	cfg.UpdatedAt = time.Now()
	h.mu.Unlock()

	// 如果状态变更，重新初始化或关闭 Provider
	if req.Status != nil {
		if *req.Status == model.ProviderStatusActive {
			if err := h.initProvider(cfg); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{
					"error": gin.H{
						"message": "Failed to initialize provider: " + err.Error(),
						"type":    "invalid_request_error",
					},
				})
				return
			}
		} else {
			// 关闭 Provider
			if p := provider.GetProvider(cfg.ID); p != nil {
				_ = p.Close()
				provider.RemoveProvider(cfg.ID)
			}
		}
	}

	cfgCopy := *cfg
	if cfgCopy.APIKey != "" {
		cfgCopy.APIKey = maskAPIKey(cfgCopy.APIKey)
	}
	c.JSON(http.StatusOK, cfgCopy)
}

// Delete 删除 Provider 配置
// DELETE /admin/providers/:id
func (h *ProviderConfigHandler) Delete(c *gin.Context) {
	id := c.Param("id")

	h.mu.Lock()
	cfg, exists := h.configs[id]
	if !exists {
		h.mu.Unlock()
		c.JSON(http.StatusNotFound, gin.H{
			"error": gin.H{"message": "Provider not found", "type": "not_found_error"},
		})
		return
	}
	delete(h.configs, id)
	h.mu.Unlock()

	// 关闭 Provider
	if p := provider.GetProvider(cfg.ID); p != nil {
		_ = p.Close()
		provider.RemoveProvider(cfg.ID)
	}

	c.JSON(http.StatusOK, gin.H{"message": "Provider deleted successfully"})
}

// Test 测试 Provider 连接
// POST /admin/providers/:id/test
func (h *ProviderConfigHandler) Test(c *gin.Context) {
	id := c.Param("id")

	h.mu.RLock()
	cfg, exists := h.configs[id]
	h.mu.RUnlock()

	if !exists {
		c.JSON(http.StatusNotFound, gin.H{
			"error": gin.H{"message": "Provider not found", "type": "not_found_error"},
		})
		return
	}

	p := provider.GetProvider(cfg.ID)
	if p == nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{"message": "Provider not initialized", "type": "invalid_request_error"},
		})
		return
	}

	if err := p.HealthCheck(c.Request.Context()); err != nil {
		c.JSON(http.StatusBadGateway, gin.H{
			"status":  "error",
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "ok",
		"message": "Provider is healthy",
		"models":  len(p.Models()),
	})
}

// GetTypes 获取支持的 Provider 类型
// GET /admin/providers/types
func (h *ProviderConfigHandler) GetTypes(c *gin.Context) {
	factories := provider.ListFactories()

	types := make([]gin.H, 0, len(factories))
	for _, name := range factories {
		types = append(types, gin.H{
			"type":        name,
			"name":        getProviderDisplayName(name),
			"description": getProviderDescription(name),
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"types": types,
	})
}

// initProvider 初始化 Provider 实例
func (h *ProviderConfigHandler) initProvider(cfg *model.ProviderConfig) error {
	providerCfg := &provider.Config{
		Name:    cfg.ID, // 使用 ID 作为实例名
		BaseURL: cfg.BaseURL,
		APIKey:  cfg.APIKey,
		Timeout: cfg.Timeout,
		Models:  cfg.Models,
		Extra:   cfg.Extra,
	}

	// 关闭旧的 Provider（如果存在）
	if old := provider.GetProvider(cfg.ID); old != nil {
		_ = old.Close()
		provider.RemoveProvider(cfg.ID)
	}

	// 使用工厂创建新的 Provider
	factory, exists := provider.GetFactory(cfg.Type)
	if !exists {
		return nil
	}

	p, err := factory(providerCfg)
	if err != nil {
		return err
	}

	provider.SetProvider(cfg.ID, p)
	return nil
}

// LoadFromConfig 从配置加载 Provider（启动时调用）
func (h *ProviderConfigHandler) LoadFromConfig(configs map[string]*model.ProviderConfig) {
	h.mu.Lock()
	defer h.mu.Unlock()

	for id, cfg := range configs {
		h.configs[id] = cfg
		if cfg.Status == model.ProviderStatusActive {
			_ = h.initProvider(cfg)
		}
	}
}

func generateID() string {
	bytes := make([]byte, 8)
	_, _ = rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

func maskAPIKey(key string) string {
	if len(key) <= 8 {
		return "****"
	}
	return key[:4] + "****" + key[len(key)-4:]
}

func getProviderDisplayName(providerType string) string {
	names := map[string]string{
		"openai":    "OpenAI",
		"anthropic": "Anthropic (Claude)",
		"dashscope": "阿里云 DashScope (通义千问)",
		"ollama":    "Ollama (本地部署)",
	}
	if name, ok := names[providerType]; ok {
		return name
	}
	return providerType
}

func getProviderDescription(providerType string) string {
	desc := map[string]string{
		"openai":    "GPT-4, GPT-3.5, DALL-E, Whisper, TTS, Embeddings",
		"anthropic": "Claude 3.5, Claude 3",
		"dashscope": "通义千问 Turbo/Plus/Max/Long, 通义万象",
		"ollama":    "Llama, Qwen, Mistral 等开源模型",
	}
	if d, ok := desc[providerType]; ok {
		return d
	}
	return ""
}

// RegisterProviderConfigRoutes 注册 Provider 配置路由
func RegisterProviderConfigRoutes(r *gin.RouterGroup, handler *ProviderConfigHandler, adminKey string) {
	admin := r.Group("/admin/providers")

	// 管理员认证
	if adminKey != "" {
		admin.Use(func(c *gin.Context) {
			authHeader := c.GetHeader("X-Admin-Key")
			if authHeader == "" {
				authHeader = c.GetHeader("Authorization")
				if len(authHeader) > 7 && authHeader[:7] == "Bearer " {
					authHeader = authHeader[7:]
				}
			}
			if authHeader != adminKey {
				c.JSON(http.StatusUnauthorized, gin.H{
					"error": gin.H{"message": "Invalid admin key", "type": "authentication_error"},
				})
				c.Abort()
				return
			}
			c.Next()
		})
	}

	admin.GET("/types", handler.GetTypes)
	admin.GET("", handler.List)
	admin.POST("", handler.Create)
	admin.GET("/:id", handler.Get)
	admin.PATCH("/:id", handler.Update)
	admin.DELETE("/:id", handler.Delete)
	admin.POST("/:id/test", handler.Test)
}
