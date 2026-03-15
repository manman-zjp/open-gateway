# AI Gateway

一个高性能、可扩展的 AI 模型 API 网关，提供统一的调用入口来访问多种大模型服务。

## 特性

- **统一 API 接口**：兼容 OpenAI API 格式，一套代码访问所有模型
- **多 Provider 支持**：OpenAI、Claude (Anthropic)、通义千问 (DashScope)、Ollama
- **全模态能力**：Chat、Vision、Image Generation、Speech-to-Text、Text-to-Speech、Embeddings
- **高性能**：基于 Go + Gin 构建，支持高并发
- **水平扩展**：无状态设计，支持多实例部署
- **可观测性**：Prometheus 指标、结构化日志、请求追踪
- **流量控制**：令牌桶限流、API Key 配额管理
- **灵活存储**：支持 MySQL、PostgreSQL 存储后端
- **缓存支持**：内存缓存、Redis 缓存

## 架构

```
┌─────────────────────────────────────────────────────────────────┐
│                         AI Gateway                               │
├─────────────────────────────────────────────────────────────────┤
│  ┌─────────┐ ┌─────────┐ ┌─────────┐ ┌─────────┐ ┌───────────┐  │
│  │ Router  │ │  Auth   │ │RateLimit│ │ Metrics │ │  Logger   │  │
│  └────┬────┘ └────┬────┘ └────┬────┘ └────┬────┘ └─────┬─────┘  │
│       └───────────┴───────────┴───────────┴───────────┘         │
│                              │                                   │
│  ┌───────────────────────────┴───────────────────────────┐      │
│  │                     Handler Layer                      │      │
│  │   ChatCompletion │ ImageGen │ Audio │ Embeddings      │      │
│  └───────────────────────────┬───────────────────────────┘      │
│                              │                                   │
│  ┌───────────────────────────┴───────────────────────────┐      │
│  │                    Provider Layer                      │      │
│  │  ┌────────┐ ┌─────────┐ ┌──────────┐ ┌────────┐       │      │
│  │  │ OpenAI │ │ Claude  │ │DashScope │ │ Ollama │       │      │
│  │  └────────┘ └─────────┘ └──────────┘ └────────┘       │      │
│  └────────────────────────────────────────────────────────┘      │
│                              │                                   │
│  ┌──────────────┬────────────┴────────────┬──────────────┐      │
│  │    Cache     │        Storage          │    Stats     │      │
│  │ Memory/Redis │    MySQL/PostgreSQL     │  Collector   │      │
│  └──────────────┴─────────────────────────┴──────────────┘      │
└─────────────────────────────────────────────────────────────────┘
```

## 快速开始

### 安装

```bash
git clone https://github.com/your-org/gateway.git
cd gateway
go build -o gateway .
```

### 配置

创建配置文件 `config.yaml`：

```yaml
server:
  host: "0.0.0.0"
  port: 8080
  read_timeout: 30s
  write_timeout: 120s
  shutdown_timeout: 10s

log:
  level: info
  format: json
  output_path: stdout

cache:
  type: memory  # memory 或 redis
  ttl: 5m
  addr: "localhost:6379"  # Redis 地址

storage:
  type: mysql  # mysql 或 postgres
  dsn: "user:password@tcp(localhost:3306)/gateway?charset=utf8mb4&parseTime=True"
  max_conns: 20
  max_idle: 10

provider:
  openai:
    enabled: true
    api_key: "sk-your-openai-key"
    base_url: "https://api.openai.com/v1"
    timeout: 60s
    models:
      - gpt-4o
      - gpt-4o-mini
      - gpt-3.5-turbo

  anthropic:
    enabled: true
    api_key: "sk-ant-your-claude-key"
    base_url: "https://api.anthropic.com"
    timeout: 120s

  dashscope:
    enabled: true
    api_key: "sk-your-dashscope-key"
    timeout: 120s
    models:
      - qwen-turbo
      - qwen-plus
      - qwen-max

  ollama:
    enabled: true
    base_url: "http://localhost:11434"
    timeout: 300s
    models:
      - llama3.2
      - qwen2.5
```

### 运行

```bash
# 使用默认配置
./gateway

# 指定配置文件
./gateway -c config.yaml

# 查看帮助
./gateway -h

# 查看版本
./gateway -v
```

### 环境变量

| 变量名 | 说明 | 默认值 |
|--------|------|--------|
| `GATEWAY_SERVER_HOST` | 服务监听地址 | 0.0.0.0 |
| `GATEWAY_SERVER_PORT` | 服务端口 | 8080 |
| `GATEWAY_LOG_LEVEL` | 日志级别 | info |
| `GATEWAY_LOG_FORMAT` | 日志格式 | json |

## 使用示例

### Chat Completion

```bash
curl http://localhost:8080/v1/chat/completions \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer your-api-key" \
  -d '{
    "model": "gpt-4o",
    "messages": [
      {"role": "user", "content": "Hello!"}
    ]
  }'
```

### 流式响应

```bash
curl http://localhost:8080/v1/chat/completions \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer your-api-key" \
  -d '{
    "model": "gpt-4o",
    "stream": true,
    "messages": [
      {"role": "user", "content": "写一首诗"}
    ]
  }'
```

### 多模态（Vision）

```bash
curl http://localhost:8080/v1/chat/completions \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer your-api-key" \
  -d '{
    "model": "gpt-4o",
    "messages": [
      {
        "role": "user",
        "content": [
          {"type": "text", "text": "这张图片里有什么？"},
          {"type": "image_url", "image_url": {"url": "https://example.com/image.jpg"}}
        ]
      }
    ]
  }'
```

### 图像生成

```bash
curl http://localhost:8080/v1/images/generations \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer your-api-key" \
  -d '{
    "model": "dall-e-3",
    "prompt": "A white cat sitting on a windowsill",
    "n": 1,
    "size": "1024x1024"
  }'
```

### Embeddings

```bash
curl http://localhost:8080/v1/embeddings \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer your-api-key" \
  -d '{
    "model": "text-embedding-3-small",
    "input": "Hello world"
  }'
```

## Provider 支持

| Provider | Chat | Vision | Image Gen | STT | TTS | Embedding |
|----------|:----:|:------:|:---------:|:---:|:---:|:---------:|
| OpenAI | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| Claude | ✅ | ✅ | - | - | - | - |
| DashScope | ✅ | ✅ | - | - | - | ✅ |
| Ollama | ✅ | ✅ | - | - | - | ✅ |

## 监控

### Prometheus 指标

访问 `/metrics` 端点获取 Prometheus 指标：

```bash
curl http://localhost:8080/metrics
```

主要指标：

| 指标名 | 类型 | 说明 |
|--------|------|------|
| `gateway_http_requests_total` | Counter | HTTP 请求总数 |
| `gateway_http_request_duration_seconds` | Histogram | 请求延迟分布 |
| `gateway_provider_requests_total` | Counter | Provider 请求数 |
| `gateway_tokens_total` | Counter | Token 使用量 |
| `gateway_active_connections` | Gauge | 活跃连接数 |
| `gateway_rate_limit_hits_total` | Counter | 限流触发次数 |

### 健康检查

```bash
# 存活检查
curl http://localhost:8080/health

# 就绪检查
curl http://localhost:8080/ready

# 服务信息
curl http://localhost:8080/info
```

## API Key 管理

### 创建 API Key

```bash
curl -X POST http://localhost:8080/admin/api-keys \
  -H "Content-Type: application/json" \
  -H "X-Admin-Key: your-admin-key" \
  -d '{
    "name": "Production Key",
    "rate_limit": 100,
    "daily_limit": 10000,
    "allowed_models": ["gpt-4o", "gpt-3.5-turbo"]
  }'
```

### 查看 API Key 使用统计

```bash
curl http://localhost:8080/admin/api-keys/{id}/usage?days=30 \
  -H "X-Admin-Key: your-admin-key"
```

## 目录结构

```
gateway/
├── apikey/           # API Key 服务
├── cache/            # 缓存抽象层
│   ├── memory/       # 内存缓存
│   └── redis/        # Redis 缓存
├── config/           # 配置加载
├── global/           # 全局变量
├── internal/
│   ├── handler/      # HTTP 处理器
│   ├── middleware/   # 中间件
│   └── router/       # 路由配置
├── model/            # 数据模型
├── pkg/
│   ├── errors/       # 错误定义
│   └── logger/       # 日志工具
├── provider/         # Provider 实现
│   ├── anthropic/    # Claude
│   ├── dashscope/    # 通义千问
│   ├── ollama/       # Ollama
│   └── openai/       # OpenAI
├── stats/            # 统计模块
├── storage/          # 存储后端
│   ├── mysql/        # MySQL
│   └── postgres/     # PostgreSQL
├── config.yaml       # 配置文件
├── main.go           # 入口
└── README.md
```

## 开发

### 添加新 Provider

1. 在 `provider/` 下创建新目录
2. 实现 `provider.Provider` 接口
3. 在 `init()` 中注册工厂函数
4. 在 `main.go` 中导入包

```go
package myprovider

import "gateway/provider"

func init() {
    provider.Register("myprovider", func(cfg *provider.Config) (provider.Provider, error) {
        return New(cfg)
    })
}
```

### 构建

```bash
# 开发构建
go build -o gateway .

# 生产构建
go build -ldflags "-s -w -X gateway/global.Version=v1.0.0" -o gateway .
```

## License

MIT License
