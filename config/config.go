package config

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/viper"
)

// Config 全局配置结构
type Config struct {
	Server   ServerConfig              `mapstructure:"server"`
	Log      LogConfig                 `mapstructure:"log"`
	Storage  StorageConfig             `mapstructure:"storage"`
	Cache    CacheConfig               `mapstructure:"cache"`
	Provider map[string]ProviderConfig `mapstructure:"provider"`
}

// ServerConfig 服务器配置
type ServerConfig struct {
	Host            string        `mapstructure:"host"`
	Port            int           `mapstructure:"port"`
	ReadTimeout     time.Duration `mapstructure:"read_timeout"`
	WriteTimeout    time.Duration `mapstructure:"write_timeout"`
	ShutdownTimeout time.Duration `mapstructure:"shutdown_timeout"`
}

// LogConfig 日志配置
type LogConfig struct {
	Level      string `mapstructure:"level"`
	Format     string `mapstructure:"format"` // json or console
	OutputPath string `mapstructure:"output_path"`
}

// StorageConfig 存储配置（支持多种后端）
type StorageConfig struct {
	Type     string        `mapstructure:"type"` // mysql, postgres, clickhouse, sqlite
	DSN      string        `mapstructure:"dsn"`
	MaxConns int           `mapstructure:"max_conns"`
	MaxIdle  int           `mapstructure:"max_idle"`
	Lifetime time.Duration `mapstructure:"lifetime"`
}

// CacheConfig 缓存配置
type CacheConfig struct {
	Type     string        `mapstructure:"type"` // memory, redis
	Addr     string        `mapstructure:"addr"`
	Password string        `mapstructure:"password"`
	DB       int           `mapstructure:"db"`
	PoolSize int           `mapstructure:"pool_size"`
	TTL      time.Duration `mapstructure:"ttl"`
}

// ProviderConfig 模型提供商配置
type ProviderConfig struct {
	Enabled bool              `mapstructure:"enabled"`
	BaseURL string            `mapstructure:"base_url"`
	APIKey  string            `mapstructure:"api_key"`
	Timeout time.Duration     `mapstructure:"timeout"`
	Models  []string          `mapstructure:"models"`
	Extra   map[string]string `mapstructure:"extra"` // 额外配置项
}

// Address 返回服务器监听地址
func (s *ServerConfig) Address() string {
	return fmt.Sprintf("%s:%d", s.Host, s.Port)
}

// Load 加载配置文件
func Load(path string) (*Config, error) {
	v := viper.New()

	// 设置默认值
	setDefaults(v)

	// 支持环境变量
	v.SetEnvPrefix("GATEWAY")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	// 加载配置文件
	if path != "" {
		v.SetConfigFile(path)
	} else {
		v.SetConfigName("config")
		v.SetConfigType("yaml")
		v.AddConfigPath(".")
		v.AddConfigPath("./config")
		v.AddConfigPath("/etc/gateway")
	}

	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("failed to read config: %w", err)
		}
		// 配置文件不存在时使用默认值
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return &cfg, nil
}

// setDefaults 设置默认配置值
func setDefaults(v *viper.Viper) {
	// Server defaults
	v.SetDefault("server.host", "0.0.0.0")
	v.SetDefault("server.port", 8080)
	v.SetDefault("server.read_timeout", "30s")
	v.SetDefault("server.write_timeout", "30s")
	v.SetDefault("server.shutdown_timeout", "10s")

	// Log defaults
	v.SetDefault("log.level", "info")
	v.SetDefault("log.format", "json")
	v.SetDefault("log.output_path", "stdout")

	// Storage defaults
	v.SetDefault("storage.type", "sqlite")
	v.SetDefault("storage.dsn", "gateway.db")
	v.SetDefault("storage.max_conns", 10)
	v.SetDefault("storage.max_idle", 5)
	v.SetDefault("storage.lifetime", "1h")

	// Cache defaults
	v.SetDefault("cache.type", "memory")
	v.SetDefault("cache.ttl", "5m")
}
