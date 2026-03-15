package anthropic

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
	defaultBaseURL   = "https://api.anthropic.com/v1"
	anthropicVersion = "2023-06-01"
)

// Provider Anthropic Claude提供商实现
type Provider struct {
	*provider.BaseProvider
	client  *http.Client
	baseURL string
	apiKey  string
}

// New 创建Anthropic提供商
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
		BaseProvider: provider.NewBaseProvider("anthropic", cfg, []provider.Capability{
			provider.CapabilityChat,
			provider.CapabilityVision,
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
			OwnedBy:  "anthropic",
			Provider: "anthropic",
		})
	}
	p.SetModels(models)
}

// ChatCompletion 聊天补全（转换协议）
func (p *Provider) ChatCompletion(ctx context.Context, req *model.ChatCompletionRequest) (*model.ChatCompletionResponse, error) {
	claudeReq := p.convertRequest(req)

	body, err := json.Marshal(claudeReq)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", p.baseURL+"/messages", bytes.NewReader(body))
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

	var claudeResp ClaudeResponse
	if err := json.NewDecoder(resp.Body).Decode(&claudeResp); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	return p.convertResponse(&claudeResp, req.Model), nil
}

// StreamChatCompletion 流式聊天补全
func (p *Provider) StreamChatCompletion(ctx context.Context, req *model.ChatCompletionRequest) (provider.Stream, error) {
	claudeReq := p.convertRequest(req)
	claudeReq.Stream = true

	body, err := json.Marshal(claudeReq)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", p.baseURL+"/messages", bytes.NewReader(body))
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

	return &ClaudeStream{
		reader: bufio.NewReader(resp.Body),
		body:   resp.Body,
		model:  req.Model,
	}, nil
}

// convertRequest 转换请求格式
func (p *Provider) convertRequest(req *model.ChatCompletionRequest) *ClaudeRequest {
	claudeReq := &ClaudeRequest{
		Model:     req.Model,
		MaxTokens: req.MaxTokens,
	}

	if claudeReq.MaxTokens == 0 {
		claudeReq.MaxTokens = 4096
	}

	if req.Temperature != nil {
		claudeReq.Temperature = req.Temperature
	}
	if req.TopP != nil {
		claudeReq.TopP = req.TopP
	}
	if len(req.Stop) > 0 {
		claudeReq.StopSequences = req.Stop
	}

	// 转换消息
	for _, msg := range req.Messages {
		if msg.Role == "system" {
			claudeReq.System = msg.GetContentString()
			continue
		}

		claudeMsg := ClaudeMessage{
			Role: msg.Role,
		}

		// 处理多模态内容
		contents := msg.GetContents()
		for _, content := range contents {
			switch content.Type {
			case model.ContentTypeText:
				claudeMsg.Content = append(claudeMsg.Content, ClaudeContent{
					Type: "text",
					Text: content.Text,
				})
			case model.ContentTypeImageURL:
				if content.ImageURL != nil {
					// 处理图片URL或base64
					url := content.ImageURL.URL
					if strings.HasPrefix(url, "data:") {
						// base64格式: data:image/png;base64,xxxx
						parts := strings.SplitN(url, ",", 2)
						if len(parts) == 2 {
							mediaType := strings.TrimPrefix(strings.Split(parts[0], ";")[0], "data:")
							claudeMsg.Content = append(claudeMsg.Content, ClaudeContent{
								Type: "image",
								Source: &ClaudeImageSource{
									Type:      "base64",
									MediaType: mediaType,
									Data:      parts[1],
								},
							})
						}
					} else {
						// URL格式
						claudeMsg.Content = append(claudeMsg.Content, ClaudeContent{
							Type: "image",
							Source: &ClaudeImageSource{
								Type: "url",
								URL:  url,
							},
						})
					}
				}
			}
		}

		claudeReq.Messages = append(claudeReq.Messages, claudeMsg)
	}

	// 转换工具
	if len(req.Tools) > 0 {
		for _, tool := range req.Tools {
			if tool.Function != nil {
				claudeReq.Tools = append(claudeReq.Tools, ClaudeTool{
					Name:        tool.Function.Name,
					Description: tool.Function.Description,
					InputSchema: tool.Function.Parameters,
				})
			}
		}
	}

	return claudeReq
}

// convertResponse 转换响应格式
func (p *Provider) convertResponse(resp *ClaudeResponse, modelName string) *model.ChatCompletionResponse {
	result := &model.ChatCompletionResponse{
		ID:      resp.ID,
		Object:  "chat.completion",
		Created: time.Now().Unix(),
		Model:   modelName,
		Usage: &model.Usage{
			PromptTokens:     resp.Usage.InputTokens,
			CompletionTokens: resp.Usage.OutputTokens,
			TotalTokens:      resp.Usage.InputTokens + resp.Usage.OutputTokens,
		},
	}

	// 转换内容
	var content string
	var toolCalls []model.ToolCall
	for _, block := range resp.Content {
		switch block.Type {
		case "text":
			content += block.Text
		case "tool_use":
			argsJSON, _ := json.Marshal(block.Input)
			toolCalls = append(toolCalls, model.ToolCall{
				ID:   block.ID,
				Type: "function",
				Function: model.FunctionCall{
					Name:      block.Name,
					Arguments: string(argsJSON),
				},
			})
		}
	}

	finishReason := "stop"
	if resp.StopReason == "tool_use" {
		finishReason = "tool_calls"
	} else if resp.StopReason == "max_tokens" {
		finishReason = "length"
	}

	result.Choices = []model.Choice{
		{
			Index: 0,
			Message: &model.Message{
				Role:      "assistant",
				Content:   content,
				ToolCalls: toolCalls,
			},
			FinishReason: finishReason,
		},
	}

	return result
}

// ImageGeneration 图像生成（不支持）
func (p *Provider) ImageGeneration(ctx context.Context, req *model.ImageGenerationRequest) (*model.ImageGenerationResponse, error) {
	return nil, provider.ErrNotImplemented("anthropic", "image_generation")
}

// SpeechToText 语音转文字（不支持）
func (p *Provider) SpeechToText(ctx context.Context, req *model.AudioTranscriptionRequest) (*model.AudioTranscriptionResponse, error) {
	return nil, provider.ErrNotImplemented("anthropic", "speech_to_text")
}

// TextToSpeech 文字转语音（不支持）
func (p *Provider) TextToSpeech(ctx context.Context, req *model.AudioSpeechRequest) (*model.AudioSpeechResponse, error) {
	return nil, provider.ErrNotImplemented("anthropic", "text_to_speech")
}

// Embeddings 向量嵌入（不支持）
func (p *Provider) Embeddings(ctx context.Context, req *model.EmbeddingRequest) (*model.EmbeddingResponse, error) {
	return nil, provider.ErrNotImplemented("anthropic", "embeddings")
}

// HealthCheck 健康检查
func (p *Provider) HealthCheck(ctx context.Context) error {
	// Claude没有专门的健康检查端点，发送一个简单请求
	return nil
}

// Close 关闭提供商
func (p *Provider) Close() error {
	p.client.CloseIdleConnections()
	return nil
}

// setHeaders 设置请求头
func (p *Provider) setHeaders(req *http.Request) {
	req.Header.Set("x-api-key", p.apiKey)
	req.Header.Set("anthropic-version", anthropicVersion)
	req.Header.Set("Content-Type", "application/json")
}

// handleError 处理错误响应
func (p *Provider) handleError(resp *http.Response) error {
	body, _ := io.ReadAll(resp.Body)

	var errResp struct {
		Error struct {
			Type    string `json:"type"`
			Message string `json:"message"`
		} `json:"error"`
	}

	if err := json.Unmarshal(body, &errResp); err == nil && errResp.Error.Message != "" {
		return fmt.Errorf("anthropic error [%s]: %s", errResp.Error.Type, errResp.Error.Message)
	}

	return fmt.Errorf("anthropic error: status %d, body: %s", resp.StatusCode, string(body))
}

// init 注册工厂
func init() {
	provider.Register("anthropic", func(cfg *provider.Config) (provider.Provider, error) {
		return New(cfg)
	})
}
