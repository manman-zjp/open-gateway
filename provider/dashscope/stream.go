package dashscope

import (
	"bufio"
	"encoding/json"
	"io"
	"strings"

	"gateway/model"
)

// Stream 通义千问流式响应
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
			return nil, err
		}

		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		if !strings.HasPrefix(line, "data: ") {
			continue
		}

		data := strings.TrimPrefix(line, "data: ")
		if data == "[DONE]" {
			return nil, io.EOF
		}

		var chunk model.StreamChunk
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			continue
		}

		return &chunk, nil
	}
}

// Close 关闭流
func (s *Stream) Close() error {
	return s.body.Close()
}
