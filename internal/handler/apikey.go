package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"gateway/apikey"
	"gateway/model"
)

// APIKeyHandler API Key 管理处理器
type APIKeyHandler struct {
	service *apikey.Service
}

// NewAPIKeyHandler 创建 API Key 处理器
func NewAPIKeyHandler(service *apikey.Service) *APIKeyHandler {
	return &APIKeyHandler{service: service}
}

// Create 创建 API Key
// POST /admin/api-keys
func (h *APIKeyHandler) Create(c *gin.Context) {
	var req model.APIKeyCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"message": "Invalid request: " + err.Error(),
				"type":    "invalid_request_error",
			},
		})
		return
	}

	resp, err := h.service.Create(c.Request.Context(), &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{
				"message": "Failed to create API key: " + err.Error(),
				"type":    "internal_error",
			},
		})
		return
	}

	c.JSON(http.StatusCreated, resp)
}

// Get 获取 API Key
// GET /admin/api-keys/:id
func (h *APIKeyHandler) Get(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"message": "API key ID is required",
				"type":    "invalid_request_error",
			},
		})
		return
	}

	key, err := h.service.Get(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": gin.H{
				"message": "API key not found",
				"type":    "not_found_error",
			},
		})
		return
	}

	c.JSON(http.StatusOK, key)
}

// List 列出 API Keys
// GET /admin/api-keys
func (h *APIKeyHandler) List(c *gin.Context) {
	var req model.APIKeyListRequest

	req.UserID = c.Query("user_id")
	if statusStr := c.Query("status"); statusStr != "" {
		status, _ := strconv.Atoi(statusStr)
		req.Status = &status
	}

	req.Limit, _ = strconv.Atoi(c.DefaultQuery("limit", "20"))
	req.Offset, _ = strconv.Atoi(c.DefaultQuery("offset", "0"))

	if req.Limit <= 0 {
		req.Limit = 20
	}
	if req.Limit > 100 {
		req.Limit = 100
	}

	resp, err := h.service.List(c.Request.Context(), &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{
				"message": "Failed to list API keys: " + err.Error(),
				"type":    "internal_error",
			},
		})
		return
	}

	c.JSON(http.StatusOK, resp)
}

// Update 更新 API Key
// PATCH /admin/api-keys/:id
func (h *APIKeyHandler) Update(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"message": "API key ID is required",
				"type":    "invalid_request_error",
			},
		})
		return
	}

	var req model.APIKeyUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"message": "Invalid request: " + err.Error(),
				"type":    "invalid_request_error",
			},
		})
		return
	}

	key, err := h.service.Update(c.Request.Context(), id, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{
				"message": "Failed to update API key: " + err.Error(),
				"type":    "internal_error",
			},
		})
		return
	}

	c.JSON(http.StatusOK, key)
}

// Delete 删除 API Key
// DELETE /admin/api-keys/:id
func (h *APIKeyHandler) Delete(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"message": "API key ID is required",
				"type":    "invalid_request_error",
			},
		})
		return
	}

	if err := h.service.Delete(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{
				"message": "Failed to delete API key: " + err.Error(),
				"type":    "internal_error",
			},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "API key deleted successfully",
	})
}

// GetUsage 获取 API Key 使用统计
// GET /admin/api-keys/:id/usage
func (h *APIKeyHandler) GetUsage(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"message": "API key ID is required",
				"type":    "invalid_request_error",
			},
		})
		return
	}

	days, _ := strconv.Atoi(c.DefaultQuery("days", "30"))
	if days <= 0 {
		days = 30
	}
	if days > 365 {
		days = 365
	}

	usage, err := h.service.GetUsage(c.Request.Context(), id, days)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{
				"message": "Failed to get usage: " + err.Error(),
				"type":    "internal_error",
			},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"key_id": id,
		"days":   days,
		"usage":  usage,
	})
}

// Revoke 吊销 API Key
// POST /admin/api-keys/:id/revoke
func (h *APIKeyHandler) Revoke(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"message": "API key ID is required",
				"type":    "invalid_request_error",
			},
		})
		return
	}

	status := model.APIKeyStatusRevoked
	_, err := h.service.Update(c.Request.Context(), id, &model.APIKeyUpdateRequest{
		Status: &status,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{
				"message": "Failed to revoke API key: " + err.Error(),
				"type":    "internal_error",
			},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "API key revoked successfully",
	})
}

// Activate 激活 API Key
// POST /admin/api-keys/:id/activate
func (h *APIKeyHandler) Activate(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"message": "API key ID is required",
				"type":    "invalid_request_error",
			},
		})
		return
	}

	status := model.APIKeyStatusActive
	_, err := h.service.Update(c.Request.Context(), id, &model.APIKeyUpdateRequest{
		Status: &status,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{
				"message": "Failed to activate API key: " + err.Error(),
				"type":    "internal_error",
			},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "API key activated successfully",
	})
}

// RegisterAPIKeyRoutes 注册 API Key 路由
func RegisterAPIKeyRoutes(r *gin.RouterGroup, handler *APIKeyHandler, adminKey string) {
	// 管理员路由
	admin := r.Group("/admin/api-keys")
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
					"error": gin.H{
						"message": "Invalid admin key",
						"type":    "authentication_error",
					},
				})
				c.Abort()
				return
			}
			c.Next()
		})
	}

	admin.POST("", handler.Create)
	admin.GET("", handler.List)
	admin.GET("/:id", handler.Get)
	admin.PATCH("/:id", handler.Update)
	admin.DELETE("/:id", handler.Delete)
	admin.GET("/:id/usage", handler.GetUsage)
	admin.POST("/:id/revoke", handler.Revoke)
	admin.POST("/:id/activate", handler.Activate)
}
