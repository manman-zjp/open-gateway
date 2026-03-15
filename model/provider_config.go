package model

import "time"

// ProviderStatus Provider 状态
type ProviderStatus int

// Provider 状态常量
const (
	ProviderStatusActive   ProviderStatus = 1
	ProviderStatusInactive ProviderStatus = 0
)

// ProviderConfig Provider 配置模型（用于动态管理）
type ProviderConfig struct {
	ID        string            `json:"id"`
	Name      string            `json:"name"` // 显示名称
	Type      string            `json:"type"` // openai, anthropic, dashscope, ollama
	BaseURL   string            `json:"base_url"`
	APIKey    string            `json:"api_key,omitempty"`
	Status    ProviderStatus    `json:"status"`
	Timeout   int               `json:"timeout"` // 秒
	Models    []string          `json:"models"`  // 启用的模型列表
	Extra     map[string]string `json:"extra,omitempty"`
	CreatedAt time.Time         `json:"created_at"`
	UpdatedAt time.Time         `json:"updated_at"`
}

// ProviderConfigCreateRequest 创建 Provider 请求
type ProviderConfigCreateRequest struct {
	Name    string            `json:"name" binding:"required"`
	Type    string            `json:"type" binding:"required"`
	BaseURL string            `json:"base_url"`
	APIKey  string            `json:"api_key"`
	Timeout int               `json:"timeout"`
	Models  []string          `json:"models"`
	Extra   map[string]string `json:"extra"`
}

// ProviderConfigUpdateRequest 更新 Provider 请求
type ProviderConfigUpdateRequest struct {
	Name    *string           `json:"name"`
	BaseURL *string           `json:"base_url"`
	APIKey  *string           `json:"api_key"`
	Status  *ProviderStatus   `json:"status"`
	Timeout *int              `json:"timeout"`
	Models  []string          `json:"models"`
	Extra   map[string]string `json:"extra"`
}

// ProviderConfigListResponse 列表响应
type ProviderConfigListResponse struct {
	Total int               `json:"total"`
	Data  []*ProviderConfig `json:"data"`
}

// ModelConfig 模型配置（用于自定义模型）
type ModelConfig struct {
	ID           string    `json:"id"`
	ProviderID   string    `json:"provider_id"`
	ModelID      string    `json:"model_id"`     // 实际模型名称
	DisplayName  string    `json:"display_name"` // 显示名称
	Capabilities []string  `json:"capabilities"` // chat, vision, image, etc.
	MaxTokens    int       `json:"max_tokens"`
	InputCost    float64   `json:"input_cost"` // 每 1K tokens 成本
	OutputCost   float64   `json:"output_cost"`
	Enabled      bool      `json:"enabled"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// ModelConfigCreateRequest 创建模型请求
type ModelConfigCreateRequest struct {
	ProviderID   string   `json:"provider_id" binding:"required"`
	ModelID      string   `json:"model_id" binding:"required"`
	DisplayName  string   `json:"display_name"`
	Capabilities []string `json:"capabilities"`
	MaxTokens    int      `json:"max_tokens"`
	InputCost    float64  `json:"input_cost"`
	OutputCost   float64  `json:"output_cost"`
}

// ModelConfigUpdateRequest 更新模型请求
type ModelConfigUpdateRequest struct {
	DisplayName  *string  `json:"display_name"`
	Capabilities []string `json:"capabilities"`
	MaxTokens    *int     `json:"max_tokens"`
	InputCost    *float64 `json:"input_cost"`
	OutputCost   *float64 `json:"output_cost"`
	Enabled      *bool    `json:"enabled"`
}
