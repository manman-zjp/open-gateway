# AI Gateway API 文档

## 概述

AI Gateway 提供兼容 OpenAI 格式的 RESTful API，所有请求使用 JSON 格式。

**Base URL**: `http://localhost:8080`

**认证方式**: Bearer Token
```
Authorization: Bearer your-api-key
```

---

## 通用响应格式

### 成功响应

```json
{
  "id": "chatcmpl-xxx",
  "object": "chat.completion",
  "created": 1234567890,
  ...
}
```

### 错误响应

```json
{
  "error": {
    "code": "invalid_request_error",
    "message": "错误描述",
    "type": "invalid_request_error",
    "details": {}
  }
}
```

### HTTP 状态码

| 状态码 | 说明 |
|--------|------|
| 200 | 成功 |
| 400 | 请求参数错误 |
| 401 | 认证失败 |
| 403 | 权限不足 |
| 404 | 资源不存在 |
| 429 | 请求过于频繁 |
| 500 | 服务器内部错误 |
| 502 | 上游服务错误 |

---

## Chat API

### POST /v1/chat/completions

创建聊天补全。

#### 请求参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| model | string | ✅ | 模型名称 |
| messages | array | ✅ | 消息数组 |
| stream | boolean | - | 是否流式返回，默认 false |
| temperature | number | - | 采样温度 0-2，默认 1 |
| top_p | number | - | 核采样概率，默认 1 |
| max_tokens | integer | - | 最大生成 token 数 |
| stop | string/array | - | 停止词 |
| presence_penalty | number | - | 话题新鲜度惩罚 -2 到 2 |
| frequency_penalty | number | - | 重复惩罚 -2 到 2 |
| tools | array | - | 可用工具列表 |
| tool_choice | string/object | - | 工具选择策略 |

#### Message 对象

```json
{
  "role": "user|assistant|system|tool",
  "content": "string 或 content 数组",
  "name": "可选名称",
  "tool_calls": [],
  "tool_call_id": "工具调用ID（role=tool时必填）"
}
```

#### Content 数组（多模态）

```json
[
  {"type": "text", "text": "文本内容"},
  {"type": "image_url", "image_url": {"url": "图片URL", "detail": "auto|low|high"}},
  {"type": "input_audio", "input_audio": {"data": "base64音频", "format": "wav|mp3"}}
]
```

#### 请求示例

**基础对话**
```bash
curl -X POST http://localhost:8080/v1/chat/completions \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer sk-xxx" \
  -d '{
    "model": "gpt-4o",
    "messages": [
      {"role": "system", "content": "你是一个有帮助的助手"},
      {"role": "user", "content": "你好"}
    ],
    "temperature": 0.7
  }'
```

**多模态（图片理解）**
```bash
curl -X POST http://localhost:8080/v1/chat/completions \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer sk-xxx" \
  -d '{
    "model": "gpt-4o",
    "messages": [
      {
        "role": "user",
        "content": [
          {"type": "text", "text": "描述这张图片"},
          {"type": "image_url", "image_url": {"url": "https://example.com/cat.jpg"}}
        ]
      }
    ]
  }'
```

**工具调用**
```bash
curl -X POST http://localhost:8080/v1/chat/completions \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer sk-xxx" \
  -d '{
    "model": "gpt-4o",
    "messages": [
      {"role": "user", "content": "北京今天天气怎么样？"}
    ],
    "tools": [
      {
        "type": "function",
        "function": {
          "name": "get_weather",
          "description": "获取指定城市的天气",
          "parameters": {
            "type": "object",
            "properties": {
              "city": {"type": "string", "description": "城市名"}
            },
            "required": ["city"]
          }
        }
      }
    ]
  }'
```

#### 响应示例

**非流式**
```json
{
  "id": "chatcmpl-abc123",
  "object": "chat.completion",
  "created": 1710500000,
  "model": "gpt-4o",
  "choices": [
    {
      "index": 0,
      "message": {
        "role": "assistant",
        "content": "你好！有什么可以帮助你的吗？"
      },
      "finish_reason": "stop"
    }
  ],
  "usage": {
    "prompt_tokens": 20,
    "completion_tokens": 15,
    "total_tokens": 35
  }
}
```

**流式 (SSE)**
```
data: {"id":"chatcmpl-abc","object":"chat.completion.chunk","created":1710500000,"model":"gpt-4o","choices":[{"index":0,"delta":{"role":"assistant"},"finish_reason":null}]}

data: {"id":"chatcmpl-abc","object":"chat.completion.chunk","created":1710500000,"model":"gpt-4o","choices":[{"index":0,"delta":{"content":"你好"},"finish_reason":null}]}

data: {"id":"chatcmpl-abc","object":"chat.completion.chunk","created":1710500000,"model":"gpt-4o","choices":[{"index":0,"delta":{},"finish_reason":"stop"}]}

data: [DONE]
```

---

## Models API

### GET /v1/models

列出所有可用模型。

#### 响应示例

```json
{
  "object": "list",
  "data": [
    {
      "id": "gpt-4o",
      "object": "model",
      "owned_by": "openai",
      "provider": "openai",
      "capabilities": ["chat", "vision", "tools"],
      "max_tokens": 128000
    },
    {
      "id": "claude-3-5-sonnet-20241022",
      "object": "model",
      "owned_by": "anthropic",
      "provider": "anthropic",
      "capabilities": ["chat", "vision", "tools"],
      "max_tokens": 200000
    }
  ]
}
```

### GET /v1/models/:model

获取指定模型信息。

```bash
curl http://localhost:8080/v1/models/gpt-4o
```

---

## Images API

### POST /v1/images/generations

生成图像。

#### 请求参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| prompt | string | ✅ | 图像描述 |
| model | string | - | 模型名称，默认 dall-e-3 |
| n | integer | - | 生成数量，默认 1 |
| size | string | - | 图像尺寸：256x256, 512x512, 1024x1024 等 |
| quality | string | - | 图像质量：standard, hd |
| style | string | - | 风格：vivid, natural |
| response_format | string | - | 返回格式：url, b64_json |

#### 请求示例

```bash
curl -X POST http://localhost:8080/v1/images/generations \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer sk-xxx" \
  -d '{
    "model": "dall-e-3",
    "prompt": "一只在窗台上晒太阳的白猫，油画风格",
    "n": 1,
    "size": "1024x1024",
    "quality": "hd"
  }'
```

#### 响应示例

```json
{
  "created": 1710500000,
  "data": [
    {
      "url": "https://example.com/image.png",
      "revised_prompt": "优化后的提示词"
    }
  ]
}
```

---

## Audio API

### POST /v1/audio/transcriptions

语音转文字 (STT)。

#### 请求参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| file | file | ✅ | 音频文件 |
| model | string | ✅ | 模型名称，如 whisper-1 |
| language | string | - | 语言代码，如 zh |
| prompt | string | - | 提示词 |
| response_format | string | - | 返回格式：json, text, srt, vtt |
| temperature | number | - | 采样温度 |

#### 请求示例

```bash
curl -X POST http://localhost:8080/v1/audio/transcriptions \
  -H "Authorization: Bearer sk-xxx" \
  -F "file=@audio.mp3" \
  -F "model=whisper-1" \
  -F "language=zh"
```

#### 响应示例

```json
{
  "text": "今天天气真好"
}
```

### POST /v1/audio/speech

文字转语音 (TTS)。

#### 请求参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| model | string | ✅ | 模型名称，如 tts-1, tts-1-hd |
| input | string | ✅ | 要转换的文本 |
| voice | string | ✅ | 声音：alloy, echo, fable, onyx, nova, shimmer |
| response_format | string | - | 格式：mp3, opus, aac, flac |
| speed | number | - | 语速 0.25-4.0 |

#### 请求示例

```bash
curl -X POST http://localhost:8080/v1/audio/speech \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer sk-xxx" \
  -d '{
    "model": "tts-1",
    "input": "你好，欢迎使用 AI Gateway",
    "voice": "alloy"
  }' \
  --output speech.mp3
```

---

## Embeddings API

### POST /v1/embeddings

创建文本向量嵌入。

#### 请求参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| model | string | ✅ | 模型名称 |
| input | string/array | ✅ | 输入文本或文本数组 |
| encoding_format | string | - | 编码格式：float, base64 |
| dimensions | integer | - | 输出维度（部分模型支持） |

#### 请求示例

```bash
curl -X POST http://localhost:8080/v1/embeddings \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer sk-xxx" \
  -d '{
    "model": "text-embedding-3-small",
    "input": ["你好世界", "Hello World"]
  }'
```

#### 响应示例

```json
{
  "object": "list",
  "data": [
    {
      "object": "embedding",
      "index": 0,
      "embedding": [0.0023, -0.0012, ...]
    },
    {
      "object": "embedding", 
      "index": 1,
      "embedding": [0.0015, -0.0008, ...]
    }
  ],
  "model": "text-embedding-3-small",
  "usage": {
    "prompt_tokens": 8,
    "total_tokens": 8
  }
}
```

---

## 管理 API

### 健康检查

#### GET /health

```json
{"status": "healthy", "version": "v0.1.0"}
```

#### GET /ready

```json
{"status": "ready", "providers": ["openai", "anthropic"]}
```

#### GET /info

```json
{
  "name": "AI Gateway",
  "version": "v0.1.0",
  "providers": ["openai", "anthropic"],
  "factories": ["openai", "anthropic", "dashscope", "ollama"]
}
```

### 统计

#### GET /v1/stats

获取请求统计。

```bash
curl http://localhost:8080/v1/stats
```

```json
{
  "total_requests": 1000,
  "success_requests": 950,
  "failed_requests": 50,
  "total_tokens": 500000,
  "prompt_tokens": 300000,
  "completion_tokens": 200000,
  "avg_latency": "1.5s"
}
```

### Prometheus 指标

#### GET /metrics

返回 Prometheus 格式的指标数据。

---

## API Key 管理

> 需要管理员密钥认证：`X-Admin-Key: your-admin-key`

### POST /admin/api-keys

创建 API Key。

#### 请求参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| name | string | ✅ | Key 名称 |
| user_id | string | - | 关联用户 ID |
| rate_limit | integer | - | 每秒请求数限制 |
| daily_limit | integer | - | 每日请求数限制 |
| monthly_limit | integer | - | 每月请求数限制 |
| allowed_models | array | - | 允许的模型列表 |
| allowed_providers | array | - | 允许的 Provider 列表 |
| expires_at | string | - | 过期时间 (ISO 8601) |
| metadata | object | - | 自定义元数据 |

#### 请求示例

```bash
curl -X POST http://localhost:8080/admin/api-keys \
  -H "Content-Type: application/json" \
  -H "X-Admin-Key: your-admin-key" \
  -d '{
    "name": "Production App Key",
    "rate_limit": 100,
    "daily_limit": 10000,
    "allowed_models": ["gpt-4o", "gpt-3.5-turbo"],
    "allowed_providers": ["openai"]
  }'
```

#### 响应示例

```json
{
  "id": "abc123def456",
  "name": "Production App Key",
  "key": "sk-xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx",
  "key_prefix": "sk-xxxxxx...",
  "created_at": "2024-03-15T10:00:00Z"
}
```

> ⚠️ `key` 字段仅在创建时返回一次，请妥善保存！

### GET /admin/api-keys

列出所有 API Key。

| 参数 | 类型 | 说明 |
|------|------|------|
| user_id | string | 按用户筛选 |
| status | integer | 按状态筛选：1=活跃, 0=禁用, -1=吊销 |
| limit | integer | 每页数量，默认 20 |
| offset | integer | 偏移量 |

```bash
curl "http://localhost:8080/admin/api-keys?limit=10" \
  -H "X-Admin-Key: your-admin-key"
```

### GET /admin/api-keys/:id

获取单个 API Key 详情。

### PATCH /admin/api-keys/:id

更新 API Key。

```bash
curl -X PATCH http://localhost:8080/admin/api-keys/abc123 \
  -H "Content-Type: application/json" \
  -H "X-Admin-Key: your-admin-key" \
  -d '{
    "name": "New Name",
    "rate_limit": 200
  }'
```

### DELETE /admin/api-keys/:id

删除 API Key。

### POST /admin/api-keys/:id/revoke

吊销 API Key（设为不可用状态）。

### POST /admin/api-keys/:id/activate

激活 API Key。

### GET /admin/api-keys/:id/usage

获取 API Key 使用统计。

| 参数 | 类型 | 说明 |
|------|------|------|
| days | integer | 统计天数，默认 30，最大 365 |

```bash
curl "http://localhost:8080/admin/api-keys/abc123/usage?days=7" \
  -H "X-Admin-Key: your-admin-key"
```

```json
{
  "key_id": "abc123",
  "days": 7,
  "usage": [
    {"date": "2024-03-15", "request_count": 100, "token_count": 50000},
    {"date": "2024-03-14", "request_count": 80, "token_count": 40000}
  ]
}
```

---

## 模型列表

### OpenAI

| 模型 | 能力 | 上下文窗口 |
|------|------|-----------|
| gpt-4o | Chat, Vision, Tools | 128K |
| gpt-4o-mini | Chat, Vision, Tools | 128K |
| gpt-4-turbo | Chat, Vision, Tools | 128K |
| gpt-4 | Chat, Tools | 8K |
| gpt-3.5-turbo | Chat, Tools | 16K |
| dall-e-3 | Image Gen | - |
| whisper-1 | STT | - |
| tts-1 / tts-1-hd | TTS | - |
| text-embedding-3-small | Embedding | - |
| text-embedding-3-large | Embedding | - |

### Anthropic (Claude)

| 模型 | 能力 | 上下文窗口 |
|------|------|-----------|
| claude-3-5-sonnet-20241022 | Chat, Vision, Tools | 200K |
| claude-3-5-haiku-20241022 | Chat, Vision, Tools | 200K |
| claude-3-opus-20240229 | Chat, Vision, Tools | 200K |
| claude-3-sonnet-20240229 | Chat, Vision, Tools | 200K |

### DashScope (通义千问)

| 模型 | 能力 | 上下文窗口 |
|------|------|-----------|
| qwen-turbo | Chat, Tools | 128K |
| qwen-plus | Chat, Tools | 128K |
| qwen-max | Chat, Tools | 32K |
| qwen-long | Chat, Tools | 10M |
| qwen-vl-plus | Chat, Vision | 32K |
| qwen-vl-max | Chat, Vision | 32K |
| text-embedding-v3 | Embedding | - |

### Ollama

| 模型 | 能力 |
|------|------|
| llama3.2 | Chat |
| llama3.1 | Chat |
| qwen2.5 | Chat |
| llava | Chat, Vision |
| nomic-embed-text | Embedding |

---

## 错误码

| 错误码 | HTTP 状态 | 说明 |
|--------|----------|------|
| invalid_request_error | 400 | 请求参数错误 |
| authentication_error | 401 | 认证失败 |
| missing_api_key | 401 | 缺少 API Key |
| invalid_api_key | 401 | 无效的 API Key |
| authorization_error | 403 | 权限不足 |
| model_access_denied | 403 | 无权访问该模型 |
| provider_access_denied | 403 | 无权访问该 Provider |
| not_found_error | 404 | 资源不存在 |
| rate_limit_exceeded | 429 | 超出速率限制 |
| provider_unavailable | 502 | Provider 不可用 |
| internal_error | 500 | 服务器内部错误 |
