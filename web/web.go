package web

import (
	"embed"
	"io/fs"
	"net/http"

	"github.com/gin-gonic/gin"
)

//go:embed static/*
var staticFS embed.FS

// RegisterRoutes 注册 Web UI 路由
func RegisterRoutes(r *gin.Engine) {
	// 获取 static 子目录
	subFS, _ := fs.Sub(staticFS, "static")

	// 静态文件服务 - 使用 /ui/ 路径避免与 /admin API 冲突
	r.StaticFS("/ui", http.FS(subFS))

	// 根路径重定向到管理页面
	r.GET("/", func(c *gin.Context) {
		c.Redirect(http.StatusFound, "/ui/")
	})
}

// GetFS 返回嵌入的文件系统
func GetFS() embed.FS {
	return staticFS
}
