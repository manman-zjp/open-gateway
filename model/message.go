package model

// ContentType 内容类型
type ContentType string

// 内容类型常量
const (
	ContentTypeText     ContentType = "text"
	ContentTypeImageURL ContentType = "image_url"
	ContentTypeImage    ContentType = "image" // base64图片
	ContentTypeAudio    ContentType = "input_audio"
	ContentTypeVideo    ContentType = "video"
)

// Content 多模态内容（OpenAI 格式）
type Content struct {
	Type       ContentType `json:"type"`
	Text       string      `json:"text,omitempty"`
	ImageURL   *ImageURL   `json:"image_url,omitempty"`
	InputAudio *InputAudio `json:"input_audio,omitempty"`
}

// ImageURL 图片URL结构
type ImageURL struct {
	URL    string `json:"url"`              // URL或base64
	Detail string `json:"detail,omitempty"` // low, high, auto
}

// InputAudio 音频输入
type InputAudio struct {
	Data   string `json:"data"`   // base64编码
	Format string `json:"format"` // wav, mp3, etc.
}

// Message 消息结构（支持多模态）
type Message struct {
	Role       string      `json:"role"`
	Content    interface{} `json:"content"` // string 或 []Content
	Name       string      `json:"name,omitempty"`
	ToolCalls  []ToolCall  `json:"tool_calls,omitempty"` // 工具调用
	ToolCallID string      `json:"tool_call_id,omitempty"`
}

// GetContentString 获取纯文本内容
func (m *Message) GetContentString() string {
	switch v := m.Content.(type) {
	case string:
		return v
	case []interface{}:
		for _, item := range v {
			if contentMap, ok := item.(map[string]interface{}); ok {
				if contentMap["type"] == "text" {
					if text, ok := contentMap["text"].(string); ok {
						return text
					}
				}
			}
		}
	}
	return ""
}

// GetContents 获取多模态内容列表
func (m *Message) GetContents() []Content {
	switch v := m.Content.(type) {
	case string:
		return []Content{{Type: ContentTypeText, Text: v}}
	case []interface{}:
		var contents []Content
		for _, item := range v {
			if contentMap, ok := item.(map[string]interface{}); ok {
				content := Content{}
				if t, ok := contentMap["type"].(string); ok {
					content.Type = ContentType(t)
				}
				if text, ok := contentMap["text"].(string); ok {
					content.Text = text
				}
				if imgURL, ok := contentMap["image_url"].(map[string]interface{}); ok {
					content.ImageURL = &ImageURL{}
					if url, ok := imgURL["url"].(string); ok {
						content.ImageURL.URL = url
					}
					if detail, ok := imgURL["detail"].(string); ok {
						content.ImageURL.Detail = detail
					}
				}
				contents = append(contents, content)
			}
		}
		return contents
	case []Content:
		return v
	}
	return nil
}

// IsMultiModal 是否包含多模态内容
func (m *Message) IsMultiModal() bool {
	switch v := m.Content.(type) {
	case string:
		return false
	case []interface{}:
		for _, item := range v {
			if contentMap, ok := item.(map[string]interface{}); ok {
				if t, ok := contentMap["type"].(string); ok {
					if t != "text" {
						return true
					}
				}
			}
		}
	case []Content:
		for _, c := range v {
			if c.Type != ContentTypeText {
				return true
			}
		}
	}
	return false
}

// ToolCall 工具调用
type ToolCall struct {
	ID       string       `json:"id"`
	Type     string       `json:"type"` // function
	Function FunctionCall `json:"function"`
}

// FunctionCall 函数调用
type FunctionCall struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

// Tool 工具定义
type Tool struct {
	Type     string       `json:"type"` // function
	Function *FunctionDef `json:"function,omitempty"`
}

// FunctionDef 函数定义
type FunctionDef struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description,omitempty"`
	Parameters  map[string]interface{} `json:"parameters,omitempty"`
}
