package errors

import (
	"fmt"
	"net/http"
)

// ErrorCode 错误码类型
type ErrorCode int

// 定义错误码
const (
	/** 通用错误 1000-1999 **/

	ErrUnknown          ErrorCode = 1000
	ErrInvalidRequest   ErrorCode = 1001
	ErrUnauthorized     ErrorCode = 1002
	ErrForbidden        ErrorCode = 1003
	ErrNotFound         ErrorCode = 1004
	ErrMethodNotAllowed ErrorCode = 1005
	ErrTooManyRequests  ErrorCode = 1006
	ErrInternal         ErrorCode = 1007

	/** Provider相关错误 2000-2999 **/

	ErrProviderNotFound    ErrorCode = 2000
	ErrProviderUnavailable ErrorCode = 2001
	ErrProviderTimeout     ErrorCode = 2002
	ErrModelNotFound       ErrorCode = 2003
	ErrInvalidAPIKey       ErrorCode = 2004
	ErrQuotaExceeded       ErrorCode = 2005

	/** 存储相关错误 3000-3999 **/

	ErrStorageConnection ErrorCode = 3000
	ErrStorageQuery      ErrorCode = 3001
	ErrStorageWrite      ErrorCode = 3002

	/** 缓存相关错误 4000-4999 **/

	ErrCacheConnection ErrorCode = 4000
	ErrCacheMiss       ErrorCode = 4001
)

// 错误码对应的HTTP状态码
var httpStatusMap = map[ErrorCode]int{
	ErrUnknown:             http.StatusInternalServerError,
	ErrInvalidRequest:      http.StatusBadRequest,
	ErrUnauthorized:        http.StatusUnauthorized,
	ErrForbidden:           http.StatusForbidden,
	ErrNotFound:            http.StatusNotFound,
	ErrMethodNotAllowed:    http.StatusMethodNotAllowed,
	ErrTooManyRequests:     http.StatusTooManyRequests,
	ErrInternal:            http.StatusInternalServerError,
	ErrProviderNotFound:    http.StatusNotFound,
	ErrProviderUnavailable: http.StatusServiceUnavailable,
	ErrProviderTimeout:     http.StatusGatewayTimeout,
	ErrModelNotFound:       http.StatusNotFound,
	ErrInvalidAPIKey:       http.StatusUnauthorized,
	ErrQuotaExceeded:       http.StatusTooManyRequests,
	ErrStorageConnection:   http.StatusInternalServerError,
	ErrStorageQuery:        http.StatusInternalServerError,
	ErrStorageWrite:        http.StatusInternalServerError,
	ErrCacheConnection:     http.StatusInternalServerError,
	ErrCacheMiss:           http.StatusNotFound,
}

// 错误码对应的默认消息
var messageMap = map[ErrorCode]string{
	ErrUnknown:             "unknown error",
	ErrInvalidRequest:      "invalid request",
	ErrUnauthorized:        "unauthorized",
	ErrForbidden:           "forbidden",
	ErrNotFound:            "not found",
	ErrMethodNotAllowed:    "method not allowed",
	ErrTooManyRequests:     "too many requests",
	ErrInternal:            "internal server error",
	ErrProviderNotFound:    "provider not found",
	ErrProviderUnavailable: "provider unavailable",
	ErrProviderTimeout:     "provider timeout",
	ErrModelNotFound:       "model not found",
	ErrInvalidAPIKey:       "invalid api key",
	ErrQuotaExceeded:       "quota exceeded",
	ErrStorageConnection:   "storage connection error",
	ErrStorageQuery:        "storage query error",
	ErrStorageWrite:        "storage write error",
	ErrCacheConnection:     "cache connection error",
	ErrCacheMiss:           "cache miss",
}

// APIError API错误结构
type APIError struct {
	Code    ErrorCode `json:"code"`
	Message string    `json:"message"`
	Details string    `json:"details,omitempty"`
	cause   error
}

// Error 实现error接口
func (e *APIError) Error() string {
	if e.Details != "" {
		return fmt.Sprintf("[%d] %s: %s", e.Code, e.Message, e.Details)
	}
	return fmt.Sprintf("[%d] %s", e.Code, e.Message)
}

// Unwrap 返回原始错误
func (e *APIError) Unwrap() error {
	return e.cause
}

// HTTPStatus 返回对应的HTTP状态码
func (e *APIError) HTTPStatus() int {
	if status, ok := httpStatusMap[e.Code]; ok {
		return status
	}
	return http.StatusInternalServerError
}

// New 创建新的API错误
func New(code ErrorCode) *APIError {
	msg := messageMap[code]
	if msg == "" {
		msg = "unknown error"
	}
	return &APIError{
		Code:    code,
		Message: msg,
	}
}

// NewWithMessage 创建带自定义消息的API错误
func NewWithMessage(code ErrorCode, message string) *APIError {
	return &APIError{
		Code:    code,
		Message: message,
	}
}

// NewWithDetails 创建带详情的API错误
func NewWithDetails(code ErrorCode, details string) *APIError {
	msg := messageMap[code]
	if msg == "" {
		msg = "unknown error"
	}
	return &APIError{
		Code:    code,
		Message: msg,
		Details: details,
	}
}

// Wrap 包装原始错误
func Wrap(code ErrorCode, err error) *APIError {
	msg := messageMap[code]
	if msg == "" {
		msg = "unknown error"
	}
	return &APIError{
		Code:    code,
		Message: msg,
		Details: err.Error(),
		cause:   err,
	}
}

// IsAPIError 判断是否为API错误
func IsAPIError(err error) bool {
	_, ok := err.(*APIError)
	return ok
}

// AsAPIError 将错误转换为API错误
func AsAPIError(err error) *APIError {
	if apiErr, ok := err.(*APIError); ok {
		return apiErr
	}
	return Wrap(ErrInternal, err)
}
