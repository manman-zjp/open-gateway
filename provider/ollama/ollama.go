package ollama

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"gateway/model"
	"gateway/provider"
)

const (
	defaultBaseURL = "http://localhost:11434"
)

// Provider Ollama提供商实现（本地部署）
type Provider struct {
	*provider.BaseProvider
	client  *http.Client
	baseURL string
}

// New 创建Ollama提供商
func New(cfg *provider.Config) (provider.Provider, error) {
	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = defaultBaseURL
	}
	baseURL = strings.TrimSuffix(baseURL, "/")

	timeout := 120
	if cfg.Timeout > 0 {
		timeout = cfg.Timeout
	}

	p := &Provider{
		BaseProvider: provider.NewBaseProvider("ollama", cfg, []provider.Capability{
			provider.CapabilityChat,
			provider.CapabilityVision,
			provider.CapabilityEmbedding,
		}),
		client: &http.Client{
			Timeout: time.Duration(timeout) * time.Second,
		},
		baseURL: baseURL,
	}

	// 初始化模型列表
	p.initModels(cfg.Models)

	return p, nil
}

// initModels 初始化模型列表
func (p *Provider) initModels(configModels []string) {
	var models []model.Info
	for _, m := range configModels {
		models = append(models, model.Info{
			ID:       m,
			Object:   "model",
			OwnedBy:  "ollama",
			Provider: "ollama",
		})
	}
	p.SetModels(models)
}

// ChatCompletion 聊天补全（使用OpenAI兼容接口）
func (p *Provider) ChatCompletion(ctx context.Context, req *model.ChatCompletionRequest) (*model.ChatCompletionResponse, error) {
	// Ollama支持OpenAI兼容接口
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", p.baseURL+"/v1/chat/completions", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, p.handleError(resp)
	}

	var result model.ChatCompletionResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	return &result, nil
}

// StreamChatCompletion 流式聊天补全
func (p *Provider) StreamChatCompletion(ctx context.Context, req *model.ChatCompletionRequest) (provider.Stream, error) {
	req.Stream = true

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", p.baseURL+"/v1/chat/completions", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("do request: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		defer resp.Body.Close()
		return nil, p.handleError(resp)
	}

	return &Stream{
		reader: bufio.NewReader(resp.Body),
		body:   resp.Body,
	}, nil
}

// ImageGeneration 图像生成（不支持）
func (p *Provider) ImageGeneration(ctx context.Context, req *model.ImageGenerationRequest) (*model.ImageGenerationResponse, error) {
	return nil, provider.ErrNotImplemented("ollama", "image_generation")
}

// SpeechToText 语音转文字（不支持）
func (p *Provider) SpeechToText(ctx context.Context, req *model.AudioTranscriptionRequest) (*model.AudioTranscriptionResponse, error) {
	return nil, provider.ErrNotImplemented("ollama", "speech_to_text")
}

// TextToSpeech 文字转语音（不支持）
func (p *Provider) TextToSpeech(ctx context.Context, req *model.AudioSpeechRequest) (*model.AudioSpeechResponse, error) {
	return nil, provider.ErrNotImplemented("ollama", "text_to_speech")
}

// Embeddings 向量嵌入
func (p *Provider) Embeddings(ctx context.Context, req *model.EmbeddingRequest) (*model.EmbeddingResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", p.baseURL+"/v1/embeddings", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, p.handleError(resp)
	}

	var result model.EmbeddingResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	return &result, nil
}

// HealthCheck 健康检查
func (p *Provider) HealthCheck(ctx context.Context) error {
	httpReq, err := http.NewRequestWithContext(ctx, "GET", p.baseURL+"/api/tags", nil)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("health check failed: status %d", resp.StatusCode)
	}

	return nil
}

// Close 关闭提供商
func (p *Provider) Close() error {
	p.client.CloseIdleConnections()
	return nil
}

// handleError 处理错误响应
func (p *Provider) handleError(resp *http.Response) error {
	body, _ := io.ReadAll(resp.Body)
	return fmt.Errorf("ollama error: status %d, body: %s", resp.StatusCode, string(body))
}

// Stream Ollama流式响应
type Stream struct {
	reader *bufio.Reader
	body   io.ReadCloser
}

// Recv 接收下一个响应块
func (s *Stream) Recv() (*model.StreamChunk, error) {
	for {
		line, err := s.reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				return nil, io.EOF
			}
			return nil, fmt.Errorf("read line: %w", err)
		}

		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		if !strings.HasPrefix(line, "data: ") {
			continue
		}

		data := strings.TrimPrefix(line, "data: ")
		if data == "[DONE]" {
			return nil, io.EOF
		}

		var chunk model.StreamChunk
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			return nil, fmt.Errorf("unmarshal chunk: %w", err)
		}

		return &chunk, nil
	}
}

// Close 关闭流
func (s *Stream) Close() error {
	return s.body.Close()
}

// init 注册工厂
func init() {
	provider.Register("ollama", func(cfg *provider.Config) (provider.Provider, error) {
		return New(cfg)
	})
}
