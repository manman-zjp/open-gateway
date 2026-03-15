package model

import "io"

// AudioTranscriptionRequest 语音转文字请求 (STT/Whisper)
type AudioTranscriptionRequest struct {
	File                   io.Reader `json:"-"`                         // 音频文件
	FileName               string    `json:"-"`                         // 文件名
	Model                  string    `json:"model" binding:"required"`  // whisper-1
	Language               string    `json:"language,omitempty"`        // ISO-639-1格式
	Prompt                 string    `json:"prompt,omitempty"`          // 提示词
	ResponseFormat         string    `json:"response_format,omitempty"` // json, text, srt, vtt, verbose_json
	Temperature            float64   `json:"temperature,omitempty"`
	TimestampGranularities []string  `json:"timestamp_granularities,omitempty"` // word, segment
}

// AudioTranscriptionResponse 语音转文字响应
type AudioTranscriptionResponse struct {
	Task     string    `json:"task,omitempty"`
	Language string    `json:"language,omitempty"`
	Duration float64   `json:"duration,omitempty"`
	Text     string    `json:"text"`
	Words    []Word    `json:"words,omitempty"`
	Segments []Segment `json:"segments,omitempty"`
}

// Word 单词级别时间戳
type Word struct {
	Word  string  `json:"word"`
	Start float64 `json:"start"`
	End   float64 `json:"end"`
}

// Segment 片段级别时间戳
type Segment struct {
	ID               int     `json:"id"`
	Seek             int     `json:"seek"`
	Start            float64 `json:"start"`
	End              float64 `json:"end"`
	Text             string  `json:"text"`
	Tokens           []int   `json:"tokens"`
	Temperature      float64 `json:"temperature"`
	AvgLogprob       float64 `json:"avg_logprob"`
	CompressionRatio float64 `json:"compression_ratio"`
	NoSpeechProb     float64 `json:"no_speech_prob"`
}

// AudioTranslationRequest 语音翻译请求（翻译为英语）
type AudioTranslationRequest struct {
	File           io.Reader `json:"-"`
	FileName       string    `json:"-"`
	Model          string    `json:"model" binding:"required"`
	Prompt         string    `json:"prompt,omitempty"`
	ResponseFormat string    `json:"response_format,omitempty"`
	Temperature    float64   `json:"temperature,omitempty"`
}

// AudioSpeechRequest 文字转语音请求 (TTS)
type AudioSpeechRequest struct {
	Model          string  `json:"model" binding:"required"`  // tts-1, tts-1-hd
	Input          string  `json:"input" binding:"required"`  // 要转换的文本，最大4096字符
	Voice          string  `json:"voice" binding:"required"`  // alloy, echo, fable, onyx, nova, shimmer
	ResponseFormat string  `json:"response_format,omitempty"` // mp3, opus, aac, flac, wav, pcm
	Speed          float64 `json:"speed,omitempty"`           // 0.25 to 4.0
}

// AudioSpeechResponse 文字转语音响应
type AudioSpeechResponse struct {
	Audio       []byte `json:"-"` // 音频二进制数据
	ContentType string `json:"-"` // 内容类型
}
