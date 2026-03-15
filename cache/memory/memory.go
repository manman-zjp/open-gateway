package memory

import (
	"context"
	"sync"
	"time"

	"gateway/cache"
	"gateway/pkg/errors"
)

// item 缓存项
type item struct {
	value      []byte
	expiration int64
}

// isExpired 检查是否过期
func (i *item) isExpired() bool {
	if i.expiration == 0 {
		return false
	}
	return time.Now().UnixNano() > i.expiration
}

// Cache 内存缓存实现
type Cache struct {
	items      map[string]*item
	mu         sync.RWMutex
	defaultTTL time.Duration
	stopCh     chan struct{}
}

// New 创建内存缓存
func New(cfg *cache.Config) (*Cache, error) {
	c := &Cache{
		items:      make(map[string]*item),
		defaultTTL: cfg.TTL,
		stopCh:     make(chan struct{}),
	}

	// 启动过期清理协程
	go c.cleanupLoop()

	return c, nil
}

// Get 获取缓存值
func (c *Cache) Get(_ context.Context, key string) ([]byte, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	item, exists := c.items[key]
	if !exists {
		return nil, errors.New(errors.ErrCacheMiss)
	}

	if item.isExpired() {
		return nil, errors.New(errors.ErrCacheMiss)
	}

	return item.value, nil
}

// Set 设置缓存值
func (c *Cache) Set(_ context.Context, key string, value []byte, ttl time.Duration) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	var expiration int64
	if ttl > 0 {
		expiration = time.Now().Add(ttl).UnixNano()
	} else if c.defaultTTL > 0 {
		expiration = time.Now().Add(c.defaultTTL).UnixNano()
	}

	c.items[key] = &item{
		value:      value,
		expiration: expiration,
	}

	return nil
}

// Delete 删除缓存
func (c *Cache) Delete(_ context.Context, key string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	delete(c.items, key)
	return nil
}

// Exists 检查键是否存在
func (c *Cache) Exists(_ context.Context, key string) (bool, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	item, exists := c.items[key]
	if !exists {
		return false, nil
	}

	return !item.isExpired(), nil
}

// Close 关闭缓存
func (c *Cache) Close() error {
	close(c.stopCh)
	return nil
}

// HealthCheck 健康检查
func (c *Cache) HealthCheck(_ context.Context) error {
	return nil
}

// cleanupLoop 定期清理过期项
func (c *Cache) cleanupLoop() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			c.cleanup()
		case <-c.stopCh:
			return
		}
	}
}

// cleanup 清理过期项
func (c *Cache) cleanup() {
	c.mu.Lock()
	defer c.mu.Unlock()

	for key, item := range c.items {
		if item.isExpired() {
			delete(c.items, key)
		}
	}
}

// init 注册工厂
func init() {
	// 可以在这里注册工厂到全局注册表
}
