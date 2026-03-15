# AI Gateway 架构设计文档

## 项目概述

AI Gateway 是一个开源的统一 API 网关，用于管理和代理各大模型厂商的 API 调用，支持自部署模型。

### 核心目标

- **统一调用入口**：为各大模型厂商提供统一的 OpenAI 兼容 API
- **使用统计**：记录和分析 API 调用情况、Token 消耗、响应时间等
- **高并发处理**：基于 Go 语言实现，支持极致的高并发处理能力
- **持久化存储**：支持多种存储后端，持久化统计和日志数据
- **水平扩展**：无状态设计，支持多实例部署
- **高可用**：支持健康检查、优雅关闭、故障转移

---

## 系统架构

### 整体架构图

```
                              ┌─────────────────┐
                              │  Load Balancer  │
                              │  (Nginx/K8s)    │
                              └────────┬────────┘
                                       │
              ┌────────────────────────┼────────────────────────┐
              │                        │                        │
     ┌────────▼────────┐      ┌────────▼────────┐      ┌────────▼────────┐
     │  Gateway Node1  │      │  Gateway Node2  │      │  Gateway NodeN  │
     │   (Stateless)   │      │   (Stateless)   │      │   (Stateless)   │
     └────────┬────────┘      └────────┬────────┘      └────────┬────────┘
              │                        │                        │
              └────────────────────────┼────────────────────────┘
                                       │
        ┌──────────────────────────────┼──────────────────────────────┐
        │                              │                              │
┌───────▼───────┐              ┌───────▼───────┐              ┌───────▼───────┐
│  Config Store │              │ Cache (Redis) │              │   Database    │
│ (File/Env/    │              │  (可选集群)    │              │ (MySQL/PG/    │
│  Etcd/Consul) │              │               │              │  ClickHouse)  │
└───────────────┘              └───────────────┘              └───────────────┘
```

### 单节点内部架构

```
┌─────────────────────────────────────────────────────────────────┐
│                        Gateway Node                              │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  ┌─────────────────────────────────────────────────────────┐    │
│  │                      Router Layer                        │    │
│  │  ┌─────────┐ ┌─────────┐ ┌─────────┐ ┌─────────┐       │    │
│  │  │Recovery │ │RequestID│ │ Logger  │ │  CORS   │       │    │
│  │  └────┬────┘ └────┬────┘ └────┬────┘ └────┬────┘       │    │
│  │       └───────────┴───────────┴───────────┘             │    │
│  └────────────────────────────┬────────────────────────────┘    │
│                               │                                  │
│  ┌────────────────────────────▼────────────────────────────┐    │
│  │                     Handler Layer                        │    │
│  │  ┌──────────────┐ ┌──────────────┐ ┌──────────────┐     │    │
│  │  │ProxyHandler  │ │HealthHandler│ │ StatsHandler │     │    │
│  │  └──────┬───────┘ └──────────────┘ └──────────────┘     │    │
│  └─────────┼───────────────────────────────────────────────┘    │
│            │                                                     │
│  ┌─────────▼───────────────────────────────────────────────┐    │
│  │                    Provider Layer                        │    │
│  │  ┌────────┐ ┌──────────┐ ┌──────────┐ ┌────────┐       │    │
│  │  │ OpenAI │ │Anthropic │ │DashScope │ │ Ollama │ ...   │    │
│  │  └────────┘ └──────────┘ └──────────┘ └────────┘       │    │
│  └──────────────────────────────────────────────────────────┘    │
│                                                                  │
│  ┌──────────────────────────────────────────────────────────┐    │
│  │                   Infrastructure                          │    │
│  │  ┌─────────┐    ┌─────────┐    ┌─────────┐              │    │
│  │  │  Cache  │    │ Storage │    │  Stats  │              │    │
│  │  └─────────┘    └─────────┘    └─────────┘              │    │
│  └──────────────────────────────────────────────────────────┘    │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

---

## 目录结构

```
gateway/
├── main.go                     # 程序入口
├── go.mod                      # Go模块定义
├── config/
│   ├── config.go               # 配置结构和加载逻辑
│   └── config.example.yaml     # 配置文件示例
├── global/
│   └── global.go               # 全局变量（版本、配置等）
├── pkg/                        # 公共基础包
│   ├── logger/
│   │   └── logger.go           # 日志封装（基于zap）
│   └── errors/
│       └── errors.go           # 统一错误处理
├── internal/                   # 内部实现
│   ├── router/
│   │   └── router.go           # 路由定义
│   ├── middleware/
│   │   ├── auth.go             # 认证中间件
│   │   ├── cors.go             # CORS中间件
│   │   ├── logger.go           # 日志中间件
│   │   ├── recovery.go         # Panic恢复中间件
│   │   └── request_id.go       # 请求ID中间件
│   └── handler/
│       ├── proxy.go            # 代理处理器
│       └── health.go           # 健康检查处理器
├── provider/                   # Provider插件（核心扩展点）
│   ├── provider.go             # Provider接口定义
│   ├── registry.go             # Provider注册中心
│   ├── openai/                 # OpenAI适配器
│   ├── anthropic/              # Anthropic适配器
│   └── ollama/                 # Ollama适配器
├── model/
│   └── request.go              # 数据模型定义
├── cache/                      # 缓存模块
│   ├── cache.go                # 缓存接口定义
│   └── memory/
│       └── memory.go           # 内存缓存实现
├── storage/                    # 存储模块
│   └── storage.go              # 存储接口定义
└── docs/
    └── architecture.md         # 本文档
```

---

## 核心接口设计

### Provider 接口

所有模型厂商适配器必须实现此接口：

```go
type Provider interface {
    // Name 返回提供商名称
    Name() string
    
    // Models 返回支持的模型列表
    Models() []model.ModelInfo
    
    // ChatCompletion 聊天补全（非流式）
    ChatCompletion(ctx context.Context, req *model.ChatCompletionRequest) (*model.ChatCompletionResponse, error)
    
    // StreamChatCompletion 聊天补全（流式）
    StreamChatCompletion(ctx context.Context, req *model.ChatCompletionRequest) (Stream, error)
    
    // HealthCheck 健康检查
    HealthCheck(ctx context.Context) error
    
    // Close 关闭连接
    Close() error
}
```

### Storage 接口

存储后端必须实现此接口：

```go
type Storage interface {
    SaveRequest(ctx context.Context, record *model.RequestRecord) error
    QueryRequests(ctx context.Context, filter *QueryFilter) ([]*model.RequestRecord, error)
    GetStats(ctx context.Context, filter *StatsFilter) (*Stats, error)
    Close() error
    HealthCheck(ctx context.Context) error
    Migrate(ctx context.Context) error
}
```

### Cache 接口

缓存后端必须实现此接口：

```go
type Cache interface {
    Get(ctx context.Context, key string) ([]byte, error)
    Set(ctx context.Context, key string, value []byte, ttl time.Duration) error
    Delete(ctx context.Context, key string) error
    Exists(ctx context.Context, key string) (bool, error)
    Close() error
    HealthCheck(ctx context.Context) error
}
```

---

## API 设计

### 基础端点

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | /health | 存活探针 |
| GET | /ready | 就绪探针 |
| GET | /info | 服务信息 |

### OpenAI 兼容端点

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | /v1/chat/completions | 聊天补全 |
| GET | /v1/models | 列出模型 |
| GET | /v1/models/:model | 获取模型详情 |

---

## 设计原则

### 1. 无状态设计

- Gateway 节点本身不存储任何状态
- 所有状态（配置、统计、会话）外置到存储层
- 支持随时水平扩展和缩容

### 2. 插件化架构

- Provider、Storage、Cache 均为接口抽象
- 通过工厂模式和注册中心管理实现
- 新增适配器无需修改核心代码

### 3. 高可用设计

- 支持 K8s liveness/readiness 探针
- 优雅关闭，处理完现有请求后退出
- Provider 级别的健康检查和故障转移

### 4. 可观测性

- 结构化日志（JSON 格式）
- 请求ID全链路追踪
- 详细的性能指标统计

---

## 配置说明

### 环境变量

所有配置支持通过环境变量覆盖，前缀为 `GATEWAY_`：

```bash
GATEWAY_SERVER_HOST=0.0.0.0
GATEWAY_SERVER_PORT=8080
GATEWAY_LOG_LEVEL=info
GATEWAY_LOG_FORMAT=json
```

### 配置文件

支持 YAML/JSON/TOML 格式，查找路径：

1. 命令行指定：`-c config.yaml`
2. 当前目录：`./config.yaml`
3. config 目录：`./config/config.yaml`
4. 系统目录：`/etc/gateway/config.yaml`

---

## 扩展指南

### 添加新的 Provider

1. 在 `provider/` 下创建新目录
2. 实现 `Provider` 接口
3. 在 `init()` 中注册工厂函数

```go
func init() {
    provider.Register("my-provider", func(cfg *provider.ProviderConfig) (provider.Provider, error) {
        return NewMyProvider(cfg)
    })
}
```

### 添加新的存储后端

1. 在 `storage/` 下创建新目录
2. 实现 `Storage` 接口
3. 在配置中指定 `storage.type`

---

## 部署建议

### Docker 部署

```dockerfile
FROM golang:1.24-alpine AS builder
WORKDIR /app
COPY . .
RUN go build -o gateway .

FROM alpine:latest
COPY --from=builder /app/gateway /usr/local/bin/
EXPOSE 8080
CMD ["gateway"]
```

### Kubernetes 部署

- 使用 Deployment 管理多副本
- 配置 HPA 实现自动扩缩容
- 使用 ConfigMap 管理配置
- 使用 Secret 管理敏感信息（API Key）

---

## 版本信息

- **当前版本**: v0.1.0
- **Go 版本**: 1.24+
- **主要依赖**:
  - gin-gonic/gin: HTTP 框架
  - spf13/viper: 配置管理
  - uber-go/zap: 日志框架
