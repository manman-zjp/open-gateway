package provider

import (
	"fmt"
	"sync"

	"gateway/pkg/logger"

	"go.uber.org/zap"
)

var (
	registry    = make(map[string]Factory)
	providers   = make(map[string]Provider)
	registryMu  sync.RWMutex
	providersMu sync.RWMutex
)

// Register 注册提供商工厂
// 通常在 init() 函数中调用
func Register(name string, factory Factory) {
	registryMu.Lock()
	defer registryMu.Unlock()

	registry[name] = factory
}

// GetFactory 获取提供商工厂
func GetFactory(name string) (Factory, bool) {
	registryMu.RLock()
	defer registryMu.RUnlock()

	factory, exists := registry[name]
	return factory, exists
}

// ListFactories 列出所有已注册的工厂
func ListFactories() []string {
	registryMu.RLock()
	defer registryMu.RUnlock()

	names := make([]string, 0, len(registry))
	for name := range registry {
		names = append(names, name)
	}
	return names
}

// InitProvider 初始化并注册一个提供商实例
func InitProvider(cfg *Config) error {
	factory, exists := GetFactory(cfg.Name)
	if !exists {
		return fmt.Errorf("provider factory not found: %s", cfg.Name)
	}

	provider, err := factory(cfg)
	if err != nil {
		return fmt.Errorf("failed to create provider %s: %w", cfg.Name, err)
	}

	providersMu.Lock()
	providers[cfg.Name] = provider
	providersMu.Unlock()

	logger.Info("provider initialized", zap.String("provider", cfg.Name))
	return nil
}

// GetProvider 获取提供商实例
func GetProvider(name string) Provider {
	providersMu.RLock()
	defer providersMu.RUnlock()

	return providers[name]
}

// SetProvider 设置提供商实例
func SetProvider(name string, p Provider) {
	providersMu.Lock()
	defer providersMu.Unlock()

	providers[name] = p
}

// RemoveProvider 移除提供商实例
func RemoveProvider(name string) {
	providersMu.Lock()
	defer providersMu.Unlock()

	delete(providers, name)
}

// GetProviderByModel 根据模型名称获取提供商
func GetProviderByModel(modelName string) (Provider, bool) {
	providersMu.RLock()
	defer providersMu.RUnlock()

	for _, p := range providers {
		for _, m := range p.Models() {
			if m.ID == modelName {
				return p, true
			}
		}
	}
	return nil, false
}

// ListProviders 列出所有已初始化的提供商
func ListProviders() []string {
	providersMu.RLock()
	defer providersMu.RUnlock()

	names := make([]string, 0, len(providers))
	for name := range providers {
		names = append(names, name)
	}
	return names
}

// GetAllProviders 获取所有提供商实例
func GetAllProviders() map[string]Provider {
	providersMu.RLock()
	defer providersMu.RUnlock()

	result := make(map[string]Provider, len(providers))
	for k, v := range providers {
		result[k] = v
	}
	return result
}

// CloseAll 关闭所有提供商
func CloseAll() {
	providersMu.Lock()
	defer providersMu.Unlock()

	for name, p := range providers {
		if err := p.Close(); err != nil {
			logger.Error("failed to close provider",
				zap.String("provider", name),
				zap.Error(err))
		}
	}
	providers = make(map[string]Provider)
}
