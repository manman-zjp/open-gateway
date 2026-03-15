package dashscope

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
	defaultBaseURL    = "https://dashscope.aliyuncs.com/compatible-mode/v1"
	defaultMultiModal = "https://dashscope.aliyuncs.com/api/v1/services/aigc/multimodal-generation/generation"
	defaultEmbedding  = "https://dashscope.aliyuncs.com/api/v1/services/embeddings/text-embedding/text-embedding"
)

// Provider 通义千问提供商实现
type Provider struct {
	*provider.BaseProvider
	client  *http.Client
	baseURL string
	apiKey  string
}

// New 创建通义千问提供商
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
		BaseProvider: provider.NewBaseProvider("dashscope", cfg, []provider.Capability{
			provider.CapabilityChat,
			provider.CapabilityVision,
			provider.CapabilityEmbedding,
			provider.CapabilityTools,
		}),
		client: &http.Client{
			Timeout: time.Duration(timeout) * time.Second,
		},
		baseURL: baseURL,
		apiKey:  cfg.APIKey,
	}

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
			OwnedBy:  "alibaba",
			Provider: "dashscope",
		})
	}
	p.SetModels(models)
}

// ChatCompletion 聊天补全（兼容 OpenAI 格式）
func (p *Provider) ChatCompletion(ctx context.Context, req *model.ChatCompletionRequest) (*model.ChatCompletionResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", p.baseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	p.setHeaders(httpReq)

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

	httpReq, err := http.NewRequestWithContext(ctx, "POST", p.baseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	p.setHeaders(httpReq)

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

// Embeddings 向量嵌入
func (p *Provider) Embeddings(ctx context.Context, req *model.EmbeddingRequest) (*model.EmbeddingResponse, error) {
	if req.Model == "" {
		req.Model = "text-embedding-v3"
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", p.baseURL+"/embeddings", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	p.setHeaders(httpReq)

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

// ImageGeneration 图像生成（通义万象）
func (p *Provider) ImageGeneration(ctx context.Context, req *model.ImageGenerationRequest) (*model.ImageGenerationResponse, error) {
	return nil, provider.ErrNotImplemented("dashscope", "image_generation")
}

// SpeechToText 语音转文字
func (p *Provider) SpeechToText(ctx context.Context, req *model.AudioTranscriptionRequest) (*model.AudioTranscriptionResponse, error) {
	return nil, provider.ErrNotImplemented("dashscope", "speech_to_text")
}

// TextToSpeech 文字转语音
func (p *Provider) TextToSpeech(ctx context.Context, req *model.AudioSpeechRequest) (*model.AudioSpeechResponse, error) {
	return nil, provider.ErrNotImplemented("dashscope", "text_to_speech")
}

// HealthCheck 健康检查
func (p *Provider) HealthCheck(ctx context.Context) error {
	httpReq, err := http.NewRequestWithContext(ctx, "GET", p.baseURL+"/models", nil)
	if err != nil {
		return err
	}
	p.setHeaders(httpReq)

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("health check failed: status %d", resp.StatusCode)
	}
	return nil
}

// Close 关闭提供商
func (p *Provider) Close() error {
	return nil
}

func (p *Provider) setHeaders(req *http.Request) {
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+p.apiKey)
}

func (p *Provider) handleError(resp *http.Response) error {
	body, _ := io.ReadAll(resp.Body)
	var errResp struct {
		Error struct {
			Message string `json:"message"`
			Type    string `json:"type"`
			Code    string `json:"code"`
		} `json:"error"`
	}
	if err := json.Unmarshal(body, &errResp); err == nil && errResp.Error.Message != "" {
		return fmt.Errorf("dashscope error [%s]: %s", errResp.Error.Code, errResp.Error.Message)
	}
	return fmt.Errorf("dashscope error: status %d, body: %s", resp.StatusCode, string(body))
}

// init 注册工厂
func init() {
	provider.Register("dashscope", func(cfg *provider.Config) (provider.Provider, error) {
		return New(cfg)
	})
}
