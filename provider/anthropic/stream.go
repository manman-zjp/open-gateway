package anthropic

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	"gateway/model"
)

// ClaudeStream Claude流式响应
type ClaudeStream struct {
	reader       *bufio.Reader
	body         io.ReadCloser
	model        string
	currentID    string
	currentIndex int
	usage        *model.Usage
}

// Recv 接收下一个响应块
func (s *ClaudeStream) Recv() (*model.StreamChunk, error) {
	for {
		line, err := s.reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				return nil, io.EOF
			}
			return nil, fmt.Errorf("read line: %w", err)
		}

		line = strings.TrimSpace(line)

		// 跳过空行
		if line == "" {
			continue
		}

		// 解析SSE事件
		if strings.HasPrefix(line, "event: ") {
			eventType := strings.TrimPrefix(line, "event: ")

			// 读取data行
			dataLine, err := s.reader.ReadString('\n')
			if err != nil {
				return nil, fmt.Errorf("read data: %w", err)
			}
			dataLine = strings.TrimSpace(dataLine)

			if !strings.HasPrefix(dataLine, "data: ") {
				continue
			}

			data := strings.TrimPrefix(dataLine, "data: ")

			return s.handleEvent(eventType, data)
		}
	}
}

// handleEvent 处理Claude事件
func (s *ClaudeStream) handleEvent(eventType, data string) (*model.StreamChunk, error) {
	var event ClaudeStreamEvent
	if err := json.Unmarshal([]byte(data), &event); err != nil {
		return nil, fmt.Errorf("unmarshal event: %w", err)
	}

	switch eventType {
	case "message_start":
		if event.Message != nil {
			s.currentID = event.Message.ID
		}
		return nil, nil // 继续读取

	case "content_block_start":
		s.currentIndex = event.Index
		return nil, nil

	case "content_block_delta":
		if event.Delta != nil && event.Delta.Text != "" {
			return &model.StreamChunk{
				ID:      s.currentID,
				Object:  "chat.completion.chunk",
				Created: time.Now().Unix(),
				Model:   s.model,
				Choices: []model.Choice{
					{
						Index: s.currentIndex,
						Delta: &model.Message{
							Content: event.Delta.Text,
						},
					},
				},
			}, nil
		}
		return nil, nil

	case "content_block_stop":
		return nil, nil

	case "message_delta":
		finishReason := ""
		if event.Delta != nil {
			if event.Delta.StopReason == "end_turn" {
				finishReason = "stop"
			} else if event.Delta.StopReason == "max_tokens" {
				finishReason = "length"
			} else if event.Delta.StopReason == "tool_use" {
				finishReason = "tool_calls"
			}
		}
		if event.Usage != nil {
			s.usage = &model.Usage{
				PromptTokens:     event.Usage.InputTokens,
				CompletionTokens: event.Usage.OutputTokens,
				TotalTokens:      event.Usage.InputTokens + event.Usage.OutputTokens,
			}
		}
		if finishReason != "" {
			return &model.StreamChunk{
				ID:      s.currentID,
				Object:  "chat.completion.chunk",
				Created: time.Now().Unix(),
				Model:   s.model,
				Choices: []model.Choice{
					{
						Index:        0,
						Delta:        &model.Message{},
						FinishReason: finishReason,
					},
				},
				Usage: s.usage,
			}, nil
		}
		return nil, nil

	case "message_stop":
		return nil, io.EOF

	case "error":
		return nil, fmt.Errorf("claude stream error: %s", data)

	default:
		return nil, nil
	}
}

// Close 关闭流
func (s *ClaudeStream) Close() error {
	return s.body.Close()
}
