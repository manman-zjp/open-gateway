package redis

import (
	"context"
	"fmt"
	"time"

	"gateway/cache"
	"gateway/pkg/errors"

	"github.com/redis/go-redis/v9"
)

// Cache Redis缓存实现
type Cache struct {
	client     *redis.Client
	defaultTTL time.Duration
}

// New 创建Redis缓存
func New(cfg *cache.Config) (*Cache, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     cfg.Addr,
		Password: cfg.Password,
		DB:       cfg.DB,
		PoolSize: cfg.PoolSize,
	})

	// 测试连接
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("redis connection failed: %w", err)
	}

	return &Cache{
		client:     client,
		defaultTTL: cfg.TTL,
	}, nil
}

// Get 获取缓存值
func (c *Cache) Get(ctx context.Context, key string) ([]byte, error) {
	val, err := c.client.Get(ctx, key).Bytes()
	if err != nil {
		if err == redis.Nil {
			return nil, errors.New(errors.ErrCacheMiss)
		}
		return nil, fmt.Errorf("redis get: %w", err)
	}
	return val, nil
}

// Set 设置缓存值
func (c *Cache) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	if ttl == 0 {
		ttl = c.defaultTTL
	}

	if err := c.client.Set(ctx, key, value, ttl).Err(); err != nil {
		return fmt.Errorf("redis set: %w", err)
	}
	return nil
}

// Delete 删除缓存
func (c *Cache) Delete(ctx context.Context, key string) error {
	if err := c.client.Del(ctx, key).Err(); err != nil {
		return fmt.Errorf("redis delete: %w", err)
	}
	return nil
}

// Exists 检查键是否存在
func (c *Cache) Exists(ctx context.Context, key string) (bool, error) {
	n, err := c.client.Exists(ctx, key).Result()
	if err != nil {
		return false, fmt.Errorf("redis exists: %w", err)
	}
	return n > 0, nil
}

// Close 关闭缓存连接
func (c *Cache) Close() error {
	return c.client.Close()
}

// HealthCheck 健康检查
func (c *Cache) HealthCheck(ctx context.Context) error {
	return c.client.Ping(ctx).Err()
}

// Incr 自增
func (c *Cache) Incr(ctx context.Context, key string) (int64, error) {
	return c.client.Incr(ctx, key).Result()
}

// IncrBy 自增指定值
func (c *Cache) IncrBy(ctx context.Context, key string, value int64) (int64, error) {
	return c.client.IncrBy(ctx, key, value).Result()
}

// Expire 设置过期时间
func (c *Cache) Expire(ctx context.Context, key string, ttl time.Duration) error {
	return c.client.Expire(ctx, key, ttl).Err()
}

// HSet 哈希设置
func (c *Cache) HSet(ctx context.Context, key string, field string, value interface{}) error {
	return c.client.HSet(ctx, key, field, value).Err()
}

// HGet 哈希获取
func (c *Cache) HGet(ctx context.Context, key string, field string) (string, error) {
	return c.client.HGet(ctx, key, field).Result()
}

// HGetAll 获取所有哈希字段
func (c *Cache) HGetAll(ctx context.Context, key string) (map[string]string, error) {
	return c.client.HGetAll(ctx, key).Result()
}

// HIncrBy 哈希自增
func (c *Cache) HIncrBy(ctx context.Context, key string, field string, value int64) (int64, error) {
	return c.client.HIncrBy(ctx, key, field, value).Result()
}

// Client 返回原始Redis客户端（用于高级操作）
func (c *Cache) Client() *redis.Client {
	return c.client
}
