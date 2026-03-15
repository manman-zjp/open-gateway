package model

// Info 模型信息
type Info struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	OwnedBy string `json:"owned_by"`

	// 扩展字段
	Provider     string   `json:"provider,omitempty"`
	Capabilities []string `json:"capabilities,omitempty"` // chat, vision, image, audio, embedding
	MaxTokens    int      `json:"max_tokens,omitempty"`
	InputCost    float64  `json:"input_cost,omitempty"` // 每1K token成本
	OutputCost   float64  `json:"output_cost,omitempty"`
}

// List 模型列表响应
type List struct {
	Object string `json:"object"`
	Data   []Info `json:"data"`
}

// Capability 能力类型
type Capability string

// 能力类型常量
const (
	CapabilityChat      Capability = "chat"
	CapabilityVision    Capability = "vision"
	CapabilityImage     Capability = "image"     // 图像生成
	CapabilityAudioSTT  Capability = "audio_stt" // 语音转文字
	CapabilityAudioTTS  Capability = "audio_tts" // 文字转语音
	CapabilityEmbedding Capability = "embedding"
	CapabilityRerank    Capability = "rerank"
	CapabilityTools     Capability = "tools" // 工具调用
)
