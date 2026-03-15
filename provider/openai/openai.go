package openai

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
	defaultBaseURL = "https://api.openai.com/v1"
)

// Provider OpenAI提供商实现
type Provider struct {
	*provider.BaseProvider
	client  *http.Client
	baseURL string
	apiKey  string
}

// New 创建OpenAI提供商
func New(cfg *provider.Config) (provider.Provider, error) {
	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = defaultBaseURL
	}
	baseURL = strings.TrimSuffix(baseURL, "/")

	timeout := 60
	if cfg.Timeout > 0 {
		timeout = cfg.Timeout
	}

	p := &Provider{
		BaseProvider: provider.NewBaseProvider("openai", cfg, []provider.Capability{
			provider.CapabilityChat,
			provider.CapabilityVision,
			provider.CapabilityImage,
			provider.CapabilityAudioSTT,
			provider.CapabilityAudioTTS,
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
			OwnedBy:  "openai",
			Provider: "openai",
		})
	}
	p.SetModels(models)
}

// ChatCompletion 聊天补全
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
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			return
		}
	}(resp.Body)

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

// ImageGeneration 图像生成
func (p *Provider) ImageGeneration(ctx context.Context, req *model.ImageGenerationRequest) (*model.ImageGenerationResponse, error) {
	if req.Model == "" {
		req.Model = "dall-e-3"
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", p.baseURL+"/images/generations", bytes.NewReader(body))
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

	var result model.ImageGenerationResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	return &result, nil
}

// SpeechToText 语音转文字
func (p *Provider) SpeechToText(ctx context.Context, req *model.AudioTranscriptionRequest) (*model.AudioTranscriptionResponse, error) {
	// 构建multipart请求
	var buf bytes.Buffer
	writer := NewMultipartWriter(&buf)

	// 添加文件
	if err := writer.WriteFile("file", req.FileName, req.File); err != nil {
		return nil, fmt.Errorf("write file: %w", err)
	}

	// 添加字段
	writer.WriteField("model", req.Model)
	if req.Language != "" {
		writer.WriteField("language", req.Language)
	}
	if req.Prompt != "" {
		writer.WriteField("prompt", req.Prompt)
	}
	if req.ResponseFormat != "" {
		writer.WriteField("response_format", req.ResponseFormat)
	}
	writer.Close()

	httpReq, err := http.NewRequestWithContext(ctx, "POST", p.baseURL+"/audio/transcriptions", &buf)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	httpReq.Header.Set("Authorization", "Bearer "+p.apiKey)
	httpReq.Header.Set("Content-Type", writer.ContentType())

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, p.handleError(resp)
	}

	var result model.AudioTranscriptionResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	return &result, nil
}

// TextToSpeech 文字转语音
func (p *Provider) TextToSpeech(ctx context.Context, req *model.AudioSpeechRequest) (*model.AudioSpeechResponse, error) {
	if req.Model == "" {
		req.Model = "tts-1"
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", p.baseURL+"/audio/speech", bytes.NewReader(body))
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

	audio, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	return &model.AudioSpeechResponse{
		Audio:       audio,
		ContentType: resp.Header.Get("Content-Type"),
	}, nil
}

// Embeddings 向量嵌入
func (p *Provider) Embeddings(ctx context.Context, req *model.EmbeddingRequest) (*model.EmbeddingResponse, error) {
	if req.Model == "" {
		req.Model = "text-embedding-3-small"
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

// HealthCheck 健康检查
func (p *Provider) HealthCheck(ctx context.Context) error {
	httpReq, err := http.NewRequestWithContext(ctx, "GET", p.baseURL+"/models", nil)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	p.setHeaders(httpReq)

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

// setHeaders 设置请求头
func (p *Provider) setHeaders(req *http.Request) {
	req.Header.Set("Authorization", "Bearer "+p.apiKey)
	req.Header.Set("Content-Type", "application/json")
}

// handleError 处理错误响应
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
		return fmt.Errorf("openai error [%s]: %s", errResp.Error.Type, errResp.Error.Message)
	}

	return fmt.Errorf("openai error: status %d, body: %s", resp.StatusCode, string(body))
}

// init 注册工厂
func init() {
	provider.Register("openai", func(cfg *provider.Config) (provider.Provider, error) {
		return New(cfg)
	})
}
