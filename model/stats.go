package model

import "time"

// RequestRecord 请求记录（用于统计和存储）
type RequestRecord struct {
	ID           string        `json:"id"`
	RequestID    string        `json:"request_id"`
	RequestType  string        `json:"request_type"` // chat, image, audio, embedding
	Provider     string        `json:"provider"`
	Model        string        `json:"model"`
	Endpoint     string        `json:"endpoint"`
	Method       string        `json:"method"`
	UserID       string        `json:"user_id,omitempty"`
	APIKeyID     string        `json:"api_key_id,omitempty"`
	StatusCode   int           `json:"status_code"`
	RequestTime  time.Time     `json:"request_time"`
	ResponseTime time.Time     `json:"response_time"`
	CreatedAt    time.Time     `json:"created_at"`
	Latency      time.Duration `json:"latency"`
	Status       int           `json:"status"`
	ErrorMessage string        `json:"error_message,omitempty"`

	// Token统计
	PromptTokens     int `json:"prompt_tokens,omitempty"`
	CompletionTokens int `json:"completion_tokens,omitempty"`
	TotalTokens      int `json:"total_tokens,omitempty"`

	// 请求信息
	ClientIP  string `json:"client_ip"`
	UserAgent string `json:"user_agent"`
	Stream    bool   `json:"stream"`

	// 可选：完整请求/响应（用于调试或审计）
	RequestBody  string `json:"request_body,omitempty"`
	ResponseBody string `json:"response_body,omitempty"`

	// 多模态统计
	HasVision    bool `json:"has_vision,omitempty"`
	ImageCount   int  `json:"image_count,omitempty"`
	AudioSeconds int  `json:"audio_seconds,omitempty"`
}

// DailyStats 每日统计
type DailyStats struct {
	Date             string  `json:"date"`
	Provider         string  `json:"provider,omitempty"`
	Model            string  `json:"model,omitempty"`
	TotalRequests    int64   `json:"total_requests"`
	SuccessRequests  int64   `json:"success_requests"`
	FailedRequests   int64   `json:"failed_requests"`
	TotalTokens      int64   `json:"total_tokens"`
	PromptTokens     int64   `json:"prompt_tokens"`
	CompletionTokens int64   `json:"completion_tokens"`
	AvgLatencyMs     float64 `json:"avg_latency_ms"`
	P50LatencyMs     float64 `json:"p50_latency_ms,omitempty"`
	P95LatencyMs     float64 `json:"p95_latency_ms,omitempty"`
	P99LatencyMs     float64 `json:"p99_latency_ms,omitempty"`
}

// CostRecord 成本记录
type CostRecord struct {
	Date       string  `json:"date"`
	Provider   string  `json:"provider"`
	Model      string  `json:"model"`
	InputCost  float64 `json:"input_cost"`
	OutputCost float64 `json:"output_cost"`
	TotalCost  float64 `json:"total_cost"`
	Currency   string  `json:"currency"` // USD, CNY
}
