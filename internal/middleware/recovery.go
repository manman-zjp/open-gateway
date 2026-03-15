package middleware

import (
	"net/http"
	"runtime/debug"

	"gateway/pkg/errors"
	"gateway/pkg/logger"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// Recovery panic恢复中间件
func Recovery() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				// 获取堆栈信息
				stack := string(debug.Stack())

				logger.Error("panic recovered",
					zap.Any("error", err),
					zap.String("stack", stack),
					zap.String("path", c.Request.URL.Path),
					zap.String("method", c.Request.Method),
				)

				// 返回内部服务器错误
				apiErr := errors.New(errors.ErrInternal)
				c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
					"error": gin.H{
						"code":    apiErr.Code,
						"message": apiErr.Message,
					},
				})
			}
		}()

		c.Next()
	}
}
