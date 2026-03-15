package model

import (
	"time"
)

// APIKeyStatus API Key 状态
type APIKeyStatus int

// API Key 状态常量
const (
	APIKeyStatusActive   APIKeyStatus = 1
	APIKeyStatusInactive APIKeyStatus = 0
	APIKeyStatusRevoked  APIKeyStatus = -1
)

// APIKey API Key 模型
type APIKey struct {
	ID               string            `json:"id"`
	Name             string            `json:"name"`
	KeyHash          string            `json:"-"` // 不暴露给 API
	KeyPrefix        string            `json:"key_prefix"`
	UserID           string            `json:"user_id,omitempty"`
	Status           APIKeyStatus      `json:"status"`
	RateLimit        int               `json:"rate_limit"`        // 每秒请求数限制 (0=无限)
	DailyLimit       int               `json:"daily_limit"`       // 每日请求数限制 (0=无限)
	MonthlyLimit     int               `json:"monthly_limit"`     // 每月请求数限制 (0=无限)
	TokenQuota       int64             `json:"token_quota"`       // Token 总额度 (0=无限)
	UsedTokens       int64             `json:"used_tokens"`       // 已使用 Token
	DailyTokenQuota  int64             `json:"daily_token_quota"` // 每日 Token 额度 (0=无限)
	UsedDailyTokens  int64             `json:"used_daily_tokens"` // 当日已使用 Token
	QuotaResetAt     *time.Time        `json:"quota_reset_at"`    // 额度重置时间
	AllowedModels    []string          `json:"allowed_models"`    // 允许的模型列表 (空=全部)
	AllowedProviders []string          `json:"allowed_providers"` // 允许的提供商列表 (空=全部)
	Metadata         map[string]string `json:"metadata,omitempty"`
	ExpiresAt        *time.Time        `json:"expires_at,omitempty"`
	LastUsedAt       *time.Time        `json:"last_used_at,omitempty"`
	CreatedAt        time.Time         `json:"created_at"`
	UpdatedAt        time.Time         `json:"updated_at"`
}

// IsActive 检查 API Key 是否有效
func (k *APIKey) IsActive() bool {
	if k.Status != APIKeyStatusActive {
		return false
	}
	if k.ExpiresAt != nil && time.Now().After(*k.ExpiresAt) {
		return false
	}
	return true
}

// CanAccessModel 检查是否可以访问指定模型
func (k *APIKey) CanAccessModel(model string) bool {
	if len(k.AllowedModels) == 0 {
		return true
	}
	for _, m := range k.AllowedModels {
		if m == model || m == "*" {
			return true
		}
	}
	return false
}

// CanAccessProvider 检查是否可以访问指定提供商
func (k *APIKey) CanAccessProvider(provider string) bool {
	if len(k.AllowedProviders) == 0 {
		return true
	}
	for _, p := range k.AllowedProviders {
		if p == provider || p == "*" {
			return true
		}
	}
	return false
}

// APIKeyCreateRequest 创建 API Key 请求
type APIKeyCreateRequest struct {
	Name             string            `json:"name" binding:"required"`
	UserID           string            `json:"user_id"`
	RateLimit        int               `json:"rate_limit"`        // 每秒请求数限制
	DailyLimit       int               `json:"daily_limit"`       // 每日请求数限制
	MonthlyLimit     int               `json:"monthly_limit"`     // 每月请求数限制
	TokenQuota       int64             `json:"token_quota"`       // Token 总额度
	DailyTokenQuota  int64             `json:"daily_token_quota"` // 每日 Token 额度
	AllowedModels    []string          `json:"allowed_models"`    // 允许的模型列表
	AllowedProviders []string          `json:"allowed_providers"` // 允许的提供商列表
	Metadata         map[string]string `json:"metadata"`
	ExpiresAt        *time.Time        `json:"expires_at"`
}

// APIKeyCreateResponse 创建 API Key 响应
type APIKeyCreateResponse struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Key       string    `json:"key"` // 只在创建时返回完整 key
	KeyPrefix string    `json:"key_prefix"`
	CreatedAt time.Time `json:"created_at"`
}

// APIKeyUpdateRequest 更新 API Key 请求
type APIKeyUpdateRequest struct {
	Name             *string           `json:"name"`
	Status           *APIKeyStatus     `json:"status"`
	RateLimit        *int              `json:"rate_limit"`
	DailyLimit       *int              `json:"daily_limit"`
	MonthlyLimit     *int              `json:"monthly_limit"`
	TokenQuota       *int64            `json:"token_quota"`
	DailyTokenQuota  *int64            `json:"daily_token_quota"`
	ResetUsedTokens  bool              `json:"reset_used_tokens"` // 重置已使用 Token
	AllowedModels    []string          `json:"allowed_models"`
	AllowedProviders []string          `json:"allowed_providers"`
	Metadata         map[string]string `json:"metadata"`
	ExpiresAt        *time.Time        `json:"expires_at"`
}

// APIKeyListRequest 列表查询请求
type APIKeyListRequest struct {
	UserID string `form:"user_id"`
	Status *int   `form:"status"`
	Limit  int    `form:"limit,default=20"`
	Offset int    `form:"offset,default=0"`
}

// APIKeyListResponse 列表响应
type APIKeyListResponse struct {
	Total int       `json:"total"`
	Data  []*APIKey `json:"data"`
}

// APIKeyUsage API Key 使用统计
type APIKeyUsage struct {
	KeyID        string    `json:"key_id"`
	Date         time.Time `json:"date"`
	RequestCount int64     `json:"request_count"`
	TokenCount   int64     `json:"token_count"`
}

// HasTokenQuota 检查是否有足够的 Token 额度
func (k *APIKey) HasTokenQuota(tokens int64) bool {
	// 检查总额度
	if k.TokenQuota > 0 && k.UsedTokens+tokens > k.TokenQuota {
		return false
	}
	// 检查日额度
	if k.DailyTokenQuota > 0 && k.UsedDailyTokens+tokens > k.DailyTokenQuota {
		return false
	}
	return true
}

// RemainingTokens 获取剩余 Token 额度
func (k *APIKey) RemainingTokens() int64 {
	if k.TokenQuota <= 0 {
		return -1 // 无限
	}
	remaining := k.TokenQuota - k.UsedTokens
	if remaining < 0 {
		return 0
	}
	return remaining
}

// RemainingDailyTokens 获取剩余日额度
func (k *APIKey) RemainingDailyTokens() int64 {
	if k.DailyTokenQuota <= 0 {
		return -1 // 无限
	}
	remaining := k.DailyTokenQuota - k.UsedDailyTokens
	if remaining < 0 {
		return 0
	}
	return remaining
}
