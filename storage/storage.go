package storage

import (
	"context"
	"time"

	"gateway/model"
)

// Storage 存储接口
// 支持多种后端实现（MySQL、PostgresSQL、ClickHouse等）
type Storage interface {
	// SaveRequest 保存请求记录
	SaveRequest(ctx context.Context, record *model.RequestRecord) error

	// QueryRequests 查询请求记录
	QueryRequests(ctx context.Context, filter *QueryFilter) ([]*model.RequestRecord, error)

	// GetStats 获取统计数据
	GetStats(ctx context.Context, filter *StatsFilter) (*Stats, error)

	// Close 关闭存储连接
	Close() error

	// HealthCheck 健康检查
	HealthCheck(ctx context.Context) error

	// Migrate 执行数据库迁移
	Migrate(ctx context.Context) error
}

// QueryFilter 查询过滤器
type QueryFilter struct {
	Provider  string
	Model     string
	UserID    string
	StartTime time.Time
	EndTime   time.Time
	Status    int
	Limit     int
	Offset    int
}

// StatsFilter 统计过滤器
type StatsFilter struct {
	Provider  string
	Model     string
	UserID    string
	StartTime time.Time
	EndTime   time.Time
	GroupBy   string // provider, model, user, hour, day
}

// Stats 统计数据
type Stats struct {
	TotalRequests    int64         `json:"total_requests"`
	SuccessRequests  int64         `json:"success_requests"`
	FailedRequests   int64         `json:"failed_requests"`
	TotalTokens      int64         `json:"total_tokens"`
	PromptTokens     int64         `json:"prompt_tokens"`
	CompletionTokens int64         `json:"completion_tokens"`
	AvgLatency       time.Duration `json:"avg_latency"`
	Groups           []StatsGroup  `json:"groups,omitempty"`
}

// StatsGroup 分组统计
type StatsGroup struct {
	Key              string        `json:"key"`
	TotalRequests    int64         `json:"total_requests"`
	SuccessRequests  int64         `json:"success_requests"`
	FailedRequests   int64         `json:"failed_requests"`
	TotalTokens      int64         `json:"total_tokens"`
	PromptTokens     int64         `json:"prompt_tokens"`
	CompletionTokens int64         `json:"completion_tokens"`
	AvgLatency       time.Duration `json:"avg_latency"`
}

// Config 存储配置
type Config struct {
	Type     string
	DSN      string
	MaxConns int
	MaxIdle  int
	Lifetime time.Duration
}

// Factory 存储工厂函数
type Factory func(cfg *Config) (Storage, error)

var (
	storageInstance Storage
)

// SetStorage 设置全局存储实例
func SetStorage(s Storage) {
	storageInstance = s
}

// GetStorage 获取全局存储实例
func GetStorage() Storage {
	return storageInstance
}
