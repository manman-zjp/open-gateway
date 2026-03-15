package model

// ChatCompletionRequest 聊天补全请求（OpenAI 兼容格式）
type ChatCompletionRequest struct {
	Model            string         `json:"model" binding:"required"`
	Messages         []Message      `json:"messages" binding:"required,min=1"`
	MaxTokens        int            `json:"max_tokens,omitempty"`
	Temperature      *float64       `json:"temperature,omitempty"`
	TopP             *float64       `json:"top_p,omitempty"`
	N                int            `json:"n,omitempty"`
	Stream           bool           `json:"stream,omitempty"`
	Stop             []string       `json:"stop,omitempty"`
	PresencePenalty  float64        `json:"presence_penalty,omitempty"`
	FrequencyPenalty float64        `json:"frequency_penalty,omitempty"`
	LogitBias        map[string]int `json:"logit_bias,omitempty"`
	User             string         `json:"user,omitempty"`
	Seed             *int           `json:"seed,omitempty"`

	// 工具相关
	Tools      []Tool      `json:"tools,omitempty"`
	ToolChoice interface{} `json:"tool_choice,omitempty"` // "auto", "none", 或具体tool

	// 响应格式
	ResponseFormat *ResponseFormat `json:"response_format,omitempty"`

	// 流式选项
	StreamOptions *StreamOptions `json:"stream_options,omitempty"`
}

// ResponseFormat 响应格式
type ResponseFormat struct {
	Type       string      `json:"type"` // text, json_object, json_schema
	JSONSchema *JSONSchema `json:"json_schema,omitempty"`
}

// JSONSchema JSON Schema定义
type JSONSchema struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description,omitempty"`
	Schema      map[string]interface{} `json:"schema"`
	Strict      bool                   `json:"strict,omitempty"`
}

// StreamOptions 流式选项
type StreamOptions struct {
	IncludeUsage bool `json:"include_usage,omitempty"`
}

// ChatCompletionResponse 聊天补全响应
type ChatCompletionResponse struct {
	ID                string   `json:"id"`
	Object            string   `json:"object"`
	Created           int64    `json:"created"`
	Model             string   `json:"model"`
	Choices           []Choice `json:"choices"`
	Usage             *Usage   `json:"usage,omitempty"`
	SystemFingerprint string   `json:"system_fingerprint,omitempty"`
}

// Choice 响应选项
type Choice struct {
	Index        int       `json:"index"`
	Message      *Message  `json:"message,omitempty"`
	Delta        *Message  `json:"delta,omitempty"` // 流式响应
	FinishReason string    `json:"finish_reason,omitempty"`
	Logprobs     *Logprobs `json:"logprobs,omitempty"`
}

// Logprobs 日志概率
type Logprobs struct {
	Content []LogprobContent `json:"content,omitempty"`
}

// LogprobContent 内容日志概率
type LogprobContent struct {
	Token   string  `json:"token"`
	Logprob float64 `json:"logprob"`
	Bytes   []int   `json:"bytes,omitempty"`
}

// Usage Token使用情况
type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// StreamChunk 流式响应块
type StreamChunk struct {
	ID                string   `json:"id"`
	Object            string   `json:"object"`
	Created           int64    `json:"created"`
	Model             string   `json:"model"`
	Choices           []Choice `json:"choices"`
	Usage             *Usage   `json:"usage,omitempty"` // 最后一个chunk可能包含
	SystemFingerprint string   `json:"system_fingerprint,omitempty"`
}
