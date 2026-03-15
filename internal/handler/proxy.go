package handler

import (
	"io"
	"net/http"
	"time"

	"gateway/internal/middleware"
	"gateway/model"
	"gateway/pkg/errors"
	"gateway/pkg/logger"
	"gateway/provider"
	"gateway/stats"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// ProxyHandler 代理处理器
type ProxyHandler struct{}

// NewProxyHandler 创建代理处理器
func NewProxyHandler() *ProxyHandler {
	return &ProxyHandler{}
}

// ChatCompletion 聊天补全接口
// POST /v1/chat/completions
func (h *ProxyHandler) ChatCompletion(c *gin.Context) {
	var req model.ChatCompletionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, errors.NewWithDetails(errors.ErrInvalidRequest, err.Error()))
		return
	}

	// 根据模型查找提供商
	p, ok := provider.GetProviderByModel(req.Model)
	if !ok {
		respondError(c, errors.NewWithDetails(errors.ErrModelNotFound, req.Model))
		return
	}

	startTime := time.Now()
	requestID := middleware.GetRequestID(c)

	logger.Info("chat completion request",
		zap.String("request_id", requestID),
		zap.String("model", req.Model),
		zap.String("provider", p.Name()),
		zap.Bool("stream", req.Stream),
	)

	// 流式响应
	if req.Stream {
		h.handleStreamResponse(c, p, &req, startTime, requestID)
		return
	}

	// 非流式响应
	h.handleNormalResponse(c, p, &req, startTime, requestID)
}

// handleNormalResponse 处理非流式响应
func (h *ProxyHandler) handleNormalResponse(c *gin.Context, p provider.Provider, req *model.ChatCompletionRequest, startTime time.Time, requestID string) {
	ctx := c.Request.Context()

	resp, err := p.ChatCompletion(ctx, req)
	latency := time.Since(startTime)

	// 记录统计
	record := &model.RequestRecord{
		ID:           uuid.New().String(),
		RequestType:  "chat",
		Provider:     p.Name(),
		Model:        req.Model,
		RequestTime:  startTime,
		ResponseTime: time.Now(),
		Latency:      latency,
		RequestID:    requestID,
		ClientIP:     c.ClientIP(),
		Stream:       false,
	}

	if err != nil {
		logger.Error("chat completion failed",
			zap.String("provider", p.Name()),
			zap.String("model", req.Model),
			zap.Error(err),
		)
		record.Status = 500
		record.ErrorMessage = err.Error()
		stats.Record(record)
		respondError(c, errors.Wrap(errors.ErrProviderUnavailable, err))
		return
	}

	record.Status = 200
	if resp.Usage != nil {
		record.PromptTokens = resp.Usage.PromptTokens
		record.CompletionTokens = resp.Usage.CompletionTokens
		record.TotalTokens = resp.Usage.TotalTokens
	}
	stats.Record(record)

	logger.Info("chat completion success",
		zap.String("provider", p.Name()),
		zap.String("model", req.Model),
		zap.Duration("latency", latency),
		zap.Int("total_tokens", record.TotalTokens),
	)

	c.JSON(http.StatusOK, resp)
}

// handleStreamResponse 处理流式响应
func (h *ProxyHandler) handleStreamResponse(c *gin.Context, p provider.Provider, req *model.ChatCompletionRequest, startTime time.Time, requestID string) {
	ctx := c.Request.Context()

	stream, err := p.StreamChatCompletion(ctx, req)
	if err != nil {
		logger.Error("stream chat completion failed",
			zap.String("provider", p.Name()),
			zap.String("model", req.Model),
			zap.Error(err),
		)
		respondError(c, errors.Wrap(errors.ErrProviderUnavailable, err))
		return
	}
	defer stream.Close()

	// 设置SSE响应头
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("Transfer-Encoding", "chunked")

	c.Stream(func(w io.Writer) bool {
		chunk, err := stream.Recv()
		if err != nil {
			if err == io.EOF {
				c.SSEvent("", "[DONE]")
				return false
			}
			logger.Error("stream recv error", zap.Error(err))
			return false
		}

		c.SSEvent("", chunk)
		return true
	})

	// 记录统计
	latency := time.Since(startTime)
	stats.Record(&model.RequestRecord{
		ID:           uuid.New().String(),
		RequestType:  "chat",
		Provider:     p.Name(),
		Model:        req.Model,
		RequestTime:  startTime,
		ResponseTime: time.Now(),
		Latency:      latency,
		Status:       200,
		RequestID:    requestID,
		ClientIP:     c.ClientIP(),
		Stream:       true,
	})

	logger.Info("stream chat completion finished",
		zap.String("provider", p.Name()),
		zap.String("model", req.Model),
		zap.Duration("latency", latency),
	)
}

// ImageGeneration 图像生成接口
// POST /v1/images/generations
func (h *ProxyHandler) ImageGeneration(c *gin.Context) {
	var req model.ImageGenerationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, errors.NewWithDetails(errors.ErrInvalidRequest, err.Error()))
		return
	}

	// 查找支持图像生成的提供商
	p, ok := findProviderWithCapability(provider.CapabilityImage, req.Model)
	if !ok {
		respondError(c, errors.NewWithDetails(errors.ErrProviderNotFound, "no provider supports image generation"))
		return
	}

	startTime := time.Now()
	ctx := c.Request.Context()

	resp, err := p.ImageGeneration(ctx, &req)
	if err != nil {
		respondError(c, errors.Wrap(errors.ErrProviderUnavailable, err))
		return
	}

	// 记录统计
	stats.Record(&model.RequestRecord{
		ID:           uuid.New().String(),
		RequestType:  "image",
		Provider:     p.Name(),
		Model:        req.Model,
		RequestTime:  startTime,
		ResponseTime: time.Now(),
		Latency:      time.Since(startTime),
		Status:       200,
		RequestID:    middleware.GetRequestID(c),
		ClientIP:     c.ClientIP(),
		ImageCount:   len(resp.Data),
	})

	c.JSON(http.StatusOK, resp)
}

// Embeddings 向量嵌入接口
// POST /v1/embeddings
func (h *ProxyHandler) Embeddings(c *gin.Context) {
	var req model.EmbeddingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, errors.NewWithDetails(errors.ErrInvalidRequest, err.Error()))
		return
	}

	p, ok := findProviderWithCapability(provider.CapabilityEmbedding, req.Model)
	if !ok {
		respondError(c, errors.NewWithDetails(errors.ErrProviderNotFound, "no provider supports embeddings"))
		return
	}

	startTime := time.Now()
	ctx := c.Request.Context()

	resp, err := p.Embeddings(ctx, &req)
	if err != nil {
		respondError(c, errors.Wrap(errors.ErrProviderUnavailable, err))
		return
	}

	// 记录统计
	stats.Record(&model.RequestRecord{
		ID:           uuid.New().String(),
		RequestType:  "embedding",
		Provider:     p.Name(),
		Model:        req.Model,
		RequestTime:  startTime,
		ResponseTime: time.Now(),
		Latency:      time.Since(startTime),
		Status:       200,
		RequestID:    middleware.GetRequestID(c),
		ClientIP:     c.ClientIP(),
		TotalTokens:  resp.Usage.TotalTokens,
	})

	c.JSON(http.StatusOK, resp)
}

// AudioTranscription 语音转文字接口
// POST /v1/audio/transcriptions
func (h *ProxyHandler) AudioTranscription(c *gin.Context) {
	// 解析multipart表单
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		respondError(c, errors.NewWithDetails(errors.ErrInvalidRequest, "file is required"))
		return
	}
	defer file.Close()

	modelName := c.PostForm("model")
	if modelName == "" {
		modelName = "whisper-1"
	}

	req := &model.AudioTranscriptionRequest{
		File:           file,
		FileName:       header.Filename,
		Model:          modelName,
		Language:       c.PostForm("language"),
		Prompt:         c.PostForm("prompt"),
		ResponseFormat: c.PostForm("response_format"),
	}

	p, ok := findProviderWithCapability(provider.CapabilityAudioSTT, modelName)
	if !ok {
		respondError(c, errors.NewWithDetails(errors.ErrProviderNotFound, "no provider supports audio transcription"))
		return
	}

	startTime := time.Now()
	ctx := c.Request.Context()

	resp, err := p.SpeechToText(ctx, req)
	if err != nil {
		respondError(c, errors.Wrap(errors.ErrProviderUnavailable, err))
		return
	}

	// 记录统计
	stats.Record(&model.RequestRecord{
		ID:           uuid.New().String(),
		RequestType:  "audio_stt",
		Provider:     p.Name(),
		Model:        modelName,
		RequestTime:  startTime,
		ResponseTime: time.Now(),
		Latency:      time.Since(startTime),
		Status:       200,
		RequestID:    middleware.GetRequestID(c),
		ClientIP:     c.ClientIP(),
	})

	c.JSON(http.StatusOK, resp)
}

// AudioSpeech 文字转语音接口
// POST /v1/audio/speech
func (h *ProxyHandler) AudioSpeech(c *gin.Context) {
	var req model.AudioSpeechRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, errors.NewWithDetails(errors.ErrInvalidRequest, err.Error()))
		return
	}

	p, ok := findProviderWithCapability(provider.CapabilityAudioTTS, req.Model)
	if !ok {
		respondError(c, errors.NewWithDetails(errors.ErrProviderNotFound, "no provider supports text to speech"))
		return
	}

	startTime := time.Now()
	ctx := c.Request.Context()

	resp, err := p.TextToSpeech(ctx, &req)
	if err != nil {
		respondError(c, errors.Wrap(errors.ErrProviderUnavailable, err))
		return
	}

	// 记录统计
	stats.Record(&model.RequestRecord{
		ID:           uuid.New().String(),
		RequestType:  "audio_tts",
		Provider:     p.Name(),
		Model:        req.Model,
		RequestTime:  startTime,
		ResponseTime: time.Now(),
		Latency:      time.Since(startTime),
		Status:       200,
		RequestID:    middleware.GetRequestID(c),
		ClientIP:     c.ClientIP(),
	})

	// 返回音频数据
	c.Data(http.StatusOK, resp.ContentType, resp.Audio)
}

// ListModels 列出所有可用模型
// GET /v1/models
func (h *ProxyHandler) ListModels(c *gin.Context) {
	var models []model.Info

	providers := provider.GetAllProviders()
	for _, p := range providers {
		models = append(models, p.Models()...)
	}

	c.JSON(http.StatusOK, model.List{
		Object: "list",
		Data:   models,
	})
}

// GetModel 获取模型详情
// GET /v1/models/:model
func (h *ProxyHandler) GetModel(c *gin.Context) {
	modelName := c.Param("model")

	providers := provider.GetAllProviders()
	for _, p := range providers {
		for _, m := range p.Models() {
			if m.ID == modelName {
				c.JSON(http.StatusOK, m)
				return
			}
		}
	}

	respondError(c, errors.NewWithDetails(errors.ErrModelNotFound, modelName))
}

// GetStats 获取统计数据
// GET /v1/stats
func (h *ProxyHandler) GetStats(c *gin.Context) {
	c.JSON(http.StatusOK, stats.GetStats())
}

// findProviderWithCapability 查找具有指定能力的提供商
func findProviderWithCapability(cap provider.Capability, modelHint string) (provider.Provider, bool) {
	providers := provider.GetAllProviders()

	// 先按模型查找
	if modelHint != "" {
		if p, ok := provider.GetProviderByModel(modelHint); ok {
			return p, true
		}
	}

	// 再按能力查找
	for _, p := range providers {
		for _, c := range p.Capabilities() {
			if c == cap {
				return p, true
			}
		}
	}

	return nil, false
}

// respondError 返回错误响应
func respondError(c *gin.Context, err *errors.APIError) {
	c.JSON(err.HTTPStatus(), gin.H{
		"error": gin.H{
			"code":    err.Code,
			"message": err.Message,
			"details": err.Details,
		},
	})
}
