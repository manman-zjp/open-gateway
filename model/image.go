package model

// ImageGenerationRequest 图像生成请求
type ImageGenerationRequest struct {
	Model          string `json:"model,omitempty"` // dall-e-2, dall-e-3
	Prompt         string `json:"prompt" binding:"required"`
	N              int    `json:"n,omitempty"`               // 生成数量 1-10
	Size           string `json:"size,omitempty"`            // 256x256, 512x512, 1024x1024, etc.
	Quality        string `json:"quality,omitempty"`         // standard, hd
	Style          string `json:"style,omitempty"`           // vivid, natural
	ResponseFormat string `json:"response_format,omitempty"` // url, b64_json
	User           string `json:"user,omitempty"`
}

// ImageGenerationResponse 图像生成响应
type ImageGenerationResponse struct {
	Created int64       `json:"created"`
	Data    []ImageData `json:"data"`
}

// ImageData 图像数据
type ImageData struct {
	URL           string `json:"url,omitempty"`
	B64JSON       string `json:"b64_json,omitempty"`
	RevisedPrompt string `json:"revised_prompt,omitempty"`
}

// ImageEditRequest 图像编辑请求
type ImageEditRequest struct {
	Image          string `json:"image" binding:"required"` // base64或文件
	Mask           string `json:"mask,omitempty"`
	Prompt         string `json:"prompt" binding:"required"`
	Model          string `json:"model,omitempty"`
	N              int    `json:"n,omitempty"`
	Size           string `json:"size,omitempty"`
	ResponseFormat string `json:"response_format,omitempty"`
	User           string `json:"user,omitempty"`
}

// ImageVariationRequest 图像变体请求
type ImageVariationRequest struct {
	Image          string `json:"image" binding:"required"`
	Model          string `json:"model,omitempty"`
	N              int    `json:"n,omitempty"`
	Size           string `json:"size,omitempty"`
	ResponseFormat string `json:"response_format,omitempty"`
	User           string `json:"user,omitempty"`
}
