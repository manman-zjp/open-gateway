package cache

import (
	"context"
	"time"
)

// Cache 缓存接口
// 支持多种后端实现（内存、Redis等）
type Cache interface {
	// Get 获取缓存值
	Get(ctx context.Context, key string) ([]byte, error)

	// Set 设置缓存值
	Set(ctx context.Context, key string, value []byte, ttl time.Duration) error

	// Delete 删除缓存
	Delete(ctx context.Context, key string) error

	// Exists 检查键是否存在
	Exists(ctx context.Context, key string) (bool, error)

	// Close 关闭缓存连接
	Close() error

	// HealthCheck 健康检查
	HealthCheck(ctx context.Context) error
}

// Factory 缓存工厂函数
type Factory func(cfg *Config) (Cache, error)

// Config 缓存配置
type Config struct {
	Type     string
	Addr     string
	Password string
	DB       int
	PoolSize int
	TTL      time.Duration
}

var (
	cacheInstance Cache
)

// SetCache 设置全局缓存实例
func SetCache(c Cache) {
	cacheInstance = c
}

// GetCache 获取全局缓存实例
func GetCache() Cache {
	return cacheInstance
}
