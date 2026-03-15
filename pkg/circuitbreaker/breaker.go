package circuitbreaker

import (
	"errors"
	"sync"
	"time"
)

var (
	ErrCircuitOpen     = errors.New("circuit breaker is open")
	ErrTooManyRequests = errors.New("too many requests")
)

// State 熔断器状态
type State int

const (
	StateClosed   State = iota // 正常（闭合）
	StateOpen                  // 熔断（断开）
	StateHalfOpen              // 半开（探测）
)

func (s State) String() string {
	switch s {
	case StateClosed:
		return "closed"
	case StateOpen:
		return "open"
	case StateHalfOpen:
		return "half-open"
	default:
		return "unknown"
	}
}

// Config 熔断器配置
type Config struct {
	MaxRequests  uint32        // 半开状态允许的最大请求数
	Interval     time.Duration // 统计时间窗口（闭合状态）
	Timeout      time.Duration // 熔断持续时间
	FailureRatio float64       // 失败率阈值（0-1）
	MinRequests  uint32        // 触发熔断的最小请求数
}

// DefaultConfig 默认配置
func DefaultConfig() *Config {
	return &Config{
		MaxRequests:  5,
		Interval:     60 * time.Second,
		Timeout:      30 * time.Second,
		FailureRatio: 0.5,
		MinRequests:  10,
	}
}

// Counts 请求统计
type Counts struct {
	Requests  uint32 // 总请求数
	Successes uint32 // 成功数
	Failures  uint32 // 失败数
	Timeouts  uint32 // 超时数
}

// CircuitBreaker 熔断器
type CircuitBreaker struct {
	name   string
	config *Config

	mu         sync.Mutex
	state      State
	counts     Counts
	expiry     time.Time // 当前统计窗口到期时间
	openExpiry time.Time // 熔断状态到期时间
}

// New 创建熔断器
func New(name string, cfg *Config) *CircuitBreaker {
	if cfg == nil {
		cfg = DefaultConfig()
	}
	return &CircuitBreaker{
		name:   name,
		config: cfg,
		state:  StateClosed,
	}
}

// Execute 执行请求（带熔断保护）
func (cb *CircuitBreaker) Execute(fn func() error) error {
	if err := cb.beforeRequest(); err != nil {
		return err
	}

	err := fn()
	cb.afterRequest(err == nil)
	return err
}

// Allow 检查是否允许请求
func (cb *CircuitBreaker) Allow() error {
	return cb.beforeRequest()
}

// Success 标记请求成功
func (cb *CircuitBreaker) Success() {
	cb.afterRequest(true)
}

// Failure 标记请求失败
func (cb *CircuitBreaker) Failure() {
	cb.afterRequest(false)
}

// State 获取当前状态
func (cb *CircuitBreaker) State() State {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	return cb.currentState(time.Now())
}

// Counts 获取当前统计
func (cb *CircuitBreaker) Counts() Counts {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	return cb.counts
}

func (cb *CircuitBreaker) beforeRequest() error {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	now := time.Now()
	state := cb.currentState(now)

	switch state {
	case StateOpen:
		return ErrCircuitOpen
	case StateHalfOpen:
		if cb.counts.Requests >= cb.config.MaxRequests {
			return ErrTooManyRequests
		}
	}

	cb.counts.Requests++
	return nil
}

func (cb *CircuitBreaker) afterRequest(success bool) {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	now := time.Now()
	state := cb.currentState(now)

	if success {
		cb.onSuccess(state, now)
	} else {
		cb.onFailure(state, now)
	}
}

func (cb *CircuitBreaker) onSuccess(state State, now time.Time) {
	switch state {
	case StateClosed:
		cb.counts.Successes++
	case StateHalfOpen:
		cb.counts.Successes++
		// 半开状态达到最大请求数且全部成功，则关闭熔断
		if cb.counts.Requests >= cb.config.MaxRequests {
			cb.setState(StateClosed, now)
		}
	}
}

func (cb *CircuitBreaker) onFailure(state State, now time.Time) {
	switch state {
	case StateClosed:
		cb.counts.Failures++
		// 检查是否需要熔断
		if cb.shouldTrip() {
			cb.setState(StateOpen, now)
		}
	case StateHalfOpen:
		// 半开状态失败，立即熔断
		cb.setState(StateOpen, now)
	}
}

func (cb *CircuitBreaker) shouldTrip() bool {
	if cb.counts.Requests < cb.config.MinRequests {
		return false
	}
	ratio := float64(cb.counts.Failures) / float64(cb.counts.Requests)
	return ratio >= cb.config.FailureRatio
}

func (cb *CircuitBreaker) currentState(now time.Time) State {
	switch cb.state {
	case StateClosed:
		if !cb.expiry.IsZero() && now.After(cb.expiry) {
			cb.resetCounts()
		}
	case StateOpen:
		if now.After(cb.openExpiry) {
			cb.setState(StateHalfOpen, now)
		}
	}
	return cb.state
}

func (cb *CircuitBreaker) setState(state State, now time.Time) {
	if cb.state == state {
		return
	}

	cb.state = state
	cb.resetCounts()

	switch state {
	case StateClosed:
		cb.expiry = now.Add(cb.config.Interval)
	case StateOpen:
		cb.openExpiry = now.Add(cb.config.Timeout)
	case StateHalfOpen:
		cb.expiry = time.Time{}
	}
}

func (cb *CircuitBreaker) resetCounts() {
	cb.counts = Counts{}
}

// Manager 熔断器管理器
type Manager struct {
	mu       sync.RWMutex
	breakers map[string]*CircuitBreaker
	config   *Config
}

var (
	defaultManager *Manager
	managerOnce    sync.Once
)

// GetManager 获取全局管理器
func GetManager() *Manager {
	managerOnce.Do(func() {
		defaultManager = &Manager{
			breakers: make(map[string]*CircuitBreaker),
			config:   DefaultConfig(),
		}
	})
	return defaultManager
}

// Get 获取或创建熔断器
func (m *Manager) Get(name string) *CircuitBreaker {
	m.mu.RLock()
	if cb, ok := m.breakers[name]; ok {
		m.mu.RUnlock()
		return cb
	}
	m.mu.RUnlock()

	m.mu.Lock()
	defer m.mu.Unlock()

	if cb, ok := m.breakers[name]; ok {
		return cb
	}

	cb := New(name, m.config)
	m.breakers[name] = cb
	return cb
}

// Remove 移除熔断器
func (m *Manager) Remove(name string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.breakers, name)
}

// Stats 获取所有熔断器状态
func (m *Manager) Stats() map[string]State {
	m.mu.RLock()
	defer m.mu.RUnlock()

	stats := make(map[string]State)
	for name, cb := range m.breakers {
		stats[name] = cb.State()
	}
	return stats
}
