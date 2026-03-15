package global

import (
	"gateway/config"
)

var (
	// Config 全局配置
	Config *config.Config

	// Version 版本信息
	Version = "v0.1.0"

	// BuildTime 构建时间
	BuildTime = "unknown"

	// GitCommit Git提交哈希
	GitCommit = "unknown"
)

// SetConfig 设置全局配置
func SetConfig(cfg *config.Config) {
	Config = cfg
}

// GetConfig 获取全局配置
func GetConfig() *config.Config {
	return Config
}
