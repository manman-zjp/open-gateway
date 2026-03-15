package httpclient

import (
	"net"
	"net/http"
	"sync"
	"time"
)

// Config HTTP 客户端配置
type Config struct {
	Timeout             time.Duration // 请求超时
	MaxIdleConns        int           // 最大空闲连接数
	MaxIdleConnsPerHost int           // 每个主机最大空闲连接
	MaxConnsPerHost     int           // 每个主机最大连接数
	IdleConnTimeout     time.Duration // 空闲连接超时
	DialTimeout         time.Duration // 连接超时
	KeepAlive           time.Duration // Keep-Alive 时间
}

// DefaultConfig 默认配置（高并发优化）
func DefaultConfig() *Config {
	return &Config{
		Timeout:             60 * time.Second,
		MaxIdleConns:        1000,
		MaxIdleConnsPerHost: 100,
		MaxConnsPerHost:     200,
		IdleConnTimeout:     90 * time.Second,
		DialTimeout:         10 * time.Second,
		KeepAlive:           30 * time.Second,
	}
}

// NewClient 创建优化的 HTTP 客户端
func NewClient(cfg *Config) *http.Client {
	if cfg == nil {
		cfg = DefaultConfig()
	}

	transport := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   cfg.DialTimeout,
			KeepAlive: cfg.KeepAlive,
		}).DialContext,
		MaxIdleConns:          cfg.MaxIdleConns,
		MaxIdleConnsPerHost:   cfg.MaxIdleConnsPerHost,
		MaxConnsPerHost:       cfg.MaxConnsPerHost,
		IdleConnTimeout:       cfg.IdleConnTimeout,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		ForceAttemptHTTP2:     true,
	}

	return &http.Client{
		Timeout:   cfg.Timeout,
		Transport: transport,
	}
}

// Pool HTTP 客户端池（按 Provider 隔离）
type Pool struct {
	mu      sync.RWMutex
	clients map[string]*http.Client
	config  *Config
}

var (
	defaultPool *Pool
	once        sync.Once
)

// GetPool 获取全局客户端池
func GetPool() *Pool {
	once.Do(func() {
		defaultPool = &Pool{
			clients: make(map[string]*http.Client),
			config:  DefaultConfig(),
		}
	})
	return defaultPool
}

// Get 获取或创建指定 Provider 的客户端
func (p *Pool) Get(providerName string, timeout time.Duration) *http.Client {
	p.mu.RLock()
	if client, ok := p.clients[providerName]; ok {
		p.mu.RUnlock()
		return client
	}
	p.mu.RUnlock()

	p.mu.Lock()
	defer p.mu.Unlock()

	// 双重检查
	if client, ok := p.clients[providerName]; ok {
		return client
	}

	cfg := *p.config
	if timeout > 0 {
		cfg.Timeout = timeout
	}

	client := NewClient(&cfg)
	p.clients[providerName] = client
	return client
}

// Close 关闭指定 Provider 的客户端
func (p *Pool) Close(providerName string) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if client, ok := p.clients[providerName]; ok {
		client.CloseIdleConnections()
		delete(p.clients, providerName)
	}
}

// CloseAll 关闭所有客户端
func (p *Pool) CloseAll() {
	p.mu.Lock()
	defer p.mu.Unlock()

	for name, client := range p.clients {
		client.CloseIdleConnections()
		delete(p.clients, name)
	}
}
