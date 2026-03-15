package provider

import (
	"context"
	"io"

	"gateway/model"
)

// Capability 能力类型
type Capability string

// 能力类型常量
const (
	CapabilityChat      Capability = "chat"
	CapabilityVision    Capability = "vision"    // 图像理解
	CapabilityImage     Capability = "image"     // 图像生成
	CapabilityAudioSTT  Capability = "audio_stt" // 语音转文字
	CapabilityAudioTTS  Capability = "audio_tts" // 文字转语音
	CapabilityEmbedding Capability = "embedding"
	CapabilityRerank    Capability = "rerank"
	CapabilityTools     Capability = "tools"
)

// Provider 模型提供商接口（全模态）
type Provider interface {
	// Name 基础信息
	Name() string
	Models() []model.Info
	Capabilities() []Capability

	// ChatCompletion 文本对话（支持多模态输入）
	ChatCompletion(ctx context.Context, req *model.ChatCompletionRequest) (*model.ChatCompletionResponse, error)
	StreamChatCompletion(ctx context.Context, req *model.ChatCompletionRequest) (Stream, error)

	// ImageGeneration 图像生成
	ImageGeneration(ctx context.Context, req *model.ImageGenerationRequest) (*model.ImageGenerationResponse, error)

	// SpeechToText 音频
	SpeechToText(ctx context.Context, req *model.AudioTranscriptionRequest) (*model.AudioTranscriptionResponse, error)
	TextToSpeech(ctx context.Context, req *model.AudioSpeechRequest) (*model.AudioSpeechResponse, error)

	// Embeddings 向量嵌入
	Embeddings(ctx context.Context, req *model.EmbeddingRequest) (*model.EmbeddingResponse, error)

	// HealthCheck 生命周期
	HealthCheck(ctx context.Context) error
	Close() error
}

// Stream 流式响应接口
type Stream interface {
	Recv() (*model.StreamChunk, error)
	Close() error
}

// Factory 提供商工厂函数类型
type Factory func(cfg *Config) (Provider, error)

// Config 提供商配置
type Config struct {
	Name    string
	BaseURL string
	APIKey  string
	Timeout int
	Models  []string
	Extra   map[string]string
}

// BaseProvider 提供商基础实现（可嵌入）
type BaseProvider struct {
	name         string
	config       *Config
	capabilities []Capability
	models       []model.Info
}

// NewBaseProvider 创建基础提供商
func NewBaseProvider(name string, cfg *Config, caps []Capability) *BaseProvider {
	return &BaseProvider{
		name:         name,
		config:       cfg,
		capabilities: caps,
	}
}

// Name 返回提供商名称
func (p *BaseProvider) Name() string {
	return p.name
}

// Config 返回配置
func (p *BaseProvider) Config() *Config {
	return p.config
}

// Capabilities 返回能力列表
func (p *BaseProvider) Capabilities() []Capability {
	return p.capabilities
}

// HasCapability 检查是否具有某能力
func (p *BaseProvider) HasCapability(cap Capability) bool {
	for _, c := range p.capabilities {
		if c == cap {
			return true
		}
	}
	return false
}

// SetModels 设置模型列表
func (p *BaseProvider) SetModels(models []model.Info) {
	p.models = models
}

// Models 返回模型列表
func (p *BaseProvider) Models() []model.Info {
	return p.models
}

// NotImplementedError 未实现错误
type NotImplementedError struct {
	Provider   string
	Capability string
}

func (e *NotImplementedError) Error() string {
	return e.Provider + " does not support " + e.Capability
}

// ErrNotImplemented 创建未实现错误
func ErrNotImplemented(provider, capability string) error {
	return &NotImplementedError{Provider: provider, Capability: capability}
}

// StreamReader SSE流读取器
type StreamReader struct {
	reader io.ReadCloser
	buffer []byte
	done   bool
}

// NewStreamReader 创建流读取器
func NewStreamReader(reader io.ReadCloser) *StreamReader {
	return &StreamReader{
		reader: reader,
		buffer: make([]byte, 0),
	}
}

// Close 关闭流
func (s *StreamReader) Close() error {
	return s.reader.Close()
}
