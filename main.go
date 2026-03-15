package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"gateway/apikey"
	"gateway/cache"
	"gateway/cache/memory"
	"gateway/config"
	"gateway/global"
	"gateway/internal/handler"
	"gateway/internal/router"
	"gateway/pkg/logger"
	"gateway/provider"
	"gateway/stats"

	"go.uber.org/zap"

	// 导入Provider以触发init注册
	_ "gateway/provider/anthropic"
	_ "gateway/provider/dashscope"
	_ "gateway/provider/ollama"
	_ "gateway/provider/openai"
)

var (
	configPath string
	showHelp   bool
	version    bool
)

func init() {
	flag.StringVar(&configPath, "config", "", "config file path")
	flag.StringVar(&configPath, "c", "", "config file path (shorthand)")
	flag.BoolVar(&showHelp, "help", false, "show help")
	flag.BoolVar(&showHelp, "h", false, "show help (shorthand)")
	flag.BoolVar(&version, "version", false, "show version")
	flag.BoolVar(&version, "v", false, "show version (shorthand)")
}

func main() {
	flag.Parse()

	if showHelp {
		printHelp()
		return
	}

	if version {
		printVersion()
		return
	}

	// 加载配置
	cfg, err := config.Load(configPath)
	if err != nil {
		_, err := fmt.Fprintf(os.Stderr, "Failed to load config: %v\n", err)
		if err != nil {
			return
		}
		os.Exit(1)
	}
	global.SetConfig(cfg)

	// 初始化日志
	if err := logger.Init(&logger.Config{
		Level:      cfg.Log.Level,
		Format:     cfg.Log.Format,
		OutputPath: cfg.Log.OutputPath,
	}); err != nil {
		_, err := fmt.Fprintf(os.Stderr, "Failed to init logger: %v\n", err)
		if err != nil {
			return
		}
		os.Exit(1)
	}
	defer logger.Sync()

	logger.Info("starting AI Gateway",
		zap.String("version", global.Version),
		zap.String("config", configPath),
	)

	// 初始化统计模块
	stats.Init(nil, nil)
	defer stats.Shutdown()

	// 初始化缓存
	if err := initCache(cfg); err != nil {
		logger.Fatal("failed to init cache", zap.Error(err))
	}

	// 初始化提供商
	if err := initProviders(cfg); err != nil {
		logger.Fatal("failed to init providers", zap.Error(err))
	}

	// 创建 Provider 配置处理器
	providerConfigHandler := handler.NewProviderConfigHandler()

	// 创建 API Key 服务
	apiKeyStore := apikey.NewMemoryStore()
	apiKeyService := apikey.NewService(apiKeyStore)

	// 创建路由
	mode := "release"
	if cfg.Log.Level == "debug" {
		mode = "debug"
	}
	r := router.New(&router.Config{
		Mode:                  mode,
		EnableAuth:            false,
		EnableMetrics:         true,
		EnableWebUI:           true,
		ProviderConfigHandler: providerConfigHandler,
		APIKeyService:         apiKeyService,
	})

	// 创建HTTP服务器
	srv := &http.Server{
		Addr:         cfg.Server.Address(),
		Handler:      r,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
	}

	// 启动服务器
	go func() {
		logger.Info("server listening", zap.String("addr", srv.Addr))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("server failed", zap.Error(err))
		}
	}()

	// 等待中断信号
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	sig := <-quit

	logger.Info("received shutdown signal", zap.String("signal", sig.String()))

	// 优雅关闭
	ctx, cancel := context.WithTimeout(context.Background(), cfg.Server.ShutdownTimeout)
	defer cancel()

	// 关闭HTTP服务器
	if err := srv.Shutdown(ctx); err != nil {
		logger.Error("server shutdown error", zap.Error(err))
	}

	// 关闭提供商
	provider.CloseAll()

	// 关闭缓存
	if c := cache.GetCache(); c != nil {
		err := c.Close()
		if err != nil {
			return
		}
	}

	logger.Info("server stopped gracefully")
}

// initCache 初始化缓存
func initCache(cfg *config.Config) error {
	switch cfg.Cache.Type {
	case "memory", "":
		c, err := memory.New(&cache.Config{
			TTL: cfg.Cache.TTL,
		})
		if err != nil {
			return err
		}
		cache.SetCache(c)
	case "redis":
		// TODO: 实现Redis缓存
		logger.Warn("redis cache not implemented, using memory cache")
		c, err := memory.New(&cache.Config{
			TTL: cfg.Cache.TTL,
		})
		if err != nil {
			return err
		}
		cache.SetCache(c)
	default:
		return fmt.Errorf("unknown cache type: %s", cfg.Cache.Type)
	}

	logger.Info("cache initialized", zap.String("type", cfg.Cache.Type))
	return nil
}

// initProviders 初始化提供商
func initProviders(cfg *config.Config) error {
	for name, pcfg := range cfg.Provider {
		if !pcfg.Enabled {
			logger.Info("provider disabled", zap.String("provider", name))
			continue
		}

		timeout := 30
		if pcfg.Timeout > 0 {
			timeout = int(pcfg.Timeout / time.Second)
		}

		providerCfg := &provider.Config{
			Name:    name,
			BaseURL: pcfg.BaseURL,
			APIKey:  pcfg.APIKey,
			Timeout: timeout,
			Models:  pcfg.Models,
			Extra:   pcfg.Extra,
		}

		if err := provider.InitProvider(providerCfg); err != nil {
			logger.Warn("failed to init provider",
				zap.String("provider", name),
				zap.Error(err))
			continue
		}
	}

	logger.Info("providers initialized", zap.Strings("providers", provider.ListProviders()))
	return nil
}

func printHelp() {
	fmt.Println("AI Gateway - Unified API Gateway for LLM Providers")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  gateway [options]")
	fmt.Println()
	fmt.Println("Options:")
	fmt.Println("  -c, --config    Config file path")
	fmt.Println("  -h, --help      Show this help message")
	fmt.Println("  -v, --version   Show version information")
	fmt.Println()
	fmt.Println("Environment Variables:")
	fmt.Println("  GATEWAY_SERVER_HOST        Server host (default: 0.0.0.0)")
	fmt.Println("  GATEWAY_SERVER_PORT        Server port (default: 8080)")
	fmt.Println("  GATEWAY_LOG_LEVEL          Log level (default: info)")
	fmt.Println("  GATEWAY_LOG_FORMAT         Log format: json/console (default: json)")
}

func printVersion() {
	fmt.Printf("AI Gateway %s\n", global.Version)
	fmt.Printf("Build Time: %s\n", global.BuildTime)
	fmt.Printf("Git Commit: %s\n", global.GitCommit)
}
