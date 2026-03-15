package openai

import (
	"bytes"
	"fmt"
	"io"
	"mime/multipart"
)

// MultipartWriter multipart写入器封装
type MultipartWriter struct {
	buf    *bytes.Buffer
	writer *multipart.Writer
}

// NewMultipartWriter 创建multipart写入器
func NewMultipartWriter(buf *bytes.Buffer) *MultipartWriter {
	return &MultipartWriter{
		buf:    buf,
		writer: multipart.NewWriter(buf),
	}
}

// WriteField 写入字段
func (w *MultipartWriter) WriteField(name, value string) error {
	return w.writer.WriteField(name, value)
}

// WriteFile 写入文件
func (w *MultipartWriter) WriteFile(fieldName, fileName string, reader io.Reader) error {
	part, err := w.writer.CreateFormFile(fieldName, fileName)
	if err != nil {
		return fmt.Errorf("create form file: %w", err)
	}

	if _, err := io.Copy(part, reader); err != nil {
		return fmt.Errorf("copy file: %w", err)
	}

	return nil
}

// ContentType 返回Content-Type
func (w *MultipartWriter) ContentType() string {
	return w.writer.FormDataContentType()
}

// Close 关闭写入器
func (w *MultipartWriter) Close() error {
	return w.writer.Close()
}
