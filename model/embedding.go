package model

// EmbeddingRequest 向量嵌入请求
type EmbeddingRequest struct {
	Model          string      `json:"model" binding:"required"`  // text-embedding-3-small, text-embedding-3-large, text-embedding-ada-002
	Input          interface{} `json:"input" binding:"required"`  // string 或 []string
	EncodingFormat string      `json:"encoding_format,omitempty"` // float, base64
	Dimensions     int         `json:"dimensions,omitempty"`      // 输出维度
	User           string      `json:"user,omitempty"`
}

// GetInputStrings 获取输入字符串列表
func (r *EmbeddingRequest) GetInputStrings() []string {
	switch v := r.Input.(type) {
	case string:
		return []string{v}
	case []interface{}:
		result := make([]string, 0, len(v))
		for _, item := range v {
			if s, ok := item.(string); ok {
				result = append(result, s)
			}
		}
		return result
	case []string:
		return v
	}
	return nil
}

// EmbeddingResponse 向量嵌入响应
type EmbeddingResponse struct {
	Object string          `json:"object"` // list
	Data   []EmbeddingData `json:"data"`
	Model  string          `json:"model"`
	Usage  EmbeddingUsage  `json:"usage"`
}

// EmbeddingData 嵌入数据
type EmbeddingData struct {
	Object    string    `json:"object"` // embedding
	Index     int       `json:"index"`
	Embedding []float64 `json:"embedding"` // 或 base64 字符串
}

// EmbeddingUsage 嵌入使用情况
type EmbeddingUsage struct {
	PromptTokens int `json:"prompt_tokens"`
	TotalTokens  int `json:"total_tokens"`
}

// RerankRequest 重排序请求
type RerankRequest struct {
	Model           string   `json:"model" binding:"required"`
	Query           string   `json:"query" binding:"required"`
	Documents       []string `json:"documents" binding:"required"`
	TopN            int      `json:"top_n,omitempty"`
	ReturnDocuments bool     `json:"return_documents,omitempty"`
}

// RerankResponse 重排序响应
type RerankResponse struct {
	Model   string       `json:"model"`
	Results []RerankItem `json:"results"`
	Usage   RerankUsage  `json:"usage"`
}

// RerankItem 重排序项
type RerankItem struct {
	Index          int     `json:"index"`
	RelevanceScore float64 `json:"relevance_score"`
	Document       string  `json:"document,omitempty"`
}

// RerankUsage 重排序使用情况
type RerankUsage struct {
	TotalTokens int `json:"total_tokens"`
}
