package openai

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"gateway/model"
)

// Stream OpenAI流式响应
type Stream struct {
	reader *bufio.Reader
	body   io.ReadCloser
}

// Recv 接收下一个响应块
func (s *Stream) Recv() (*model.StreamChunk, error) {
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

		// 检查是否为SSE数据行
		if !strings.HasPrefix(line, "data: ") {
			continue
		}

		data := strings.TrimPrefix(line, "data: ")

		// 检查结束标记
		if data == "[DONE]" {
			return nil, io.EOF
		}

		// 解析JSON
		var chunk model.StreamChunk
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			return nil, fmt.Errorf("unmarshal chunk: %w", err)
		}

		return &chunk, nil
	}
}

// Close 关闭流
func (s *Stream) Close() error {
	return s.body.Close()
}
