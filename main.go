package main

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

const (
	sourceFlagKey     = "X-FileSource"
	modified          = "custom-modified file"
	original          = "proxy-original file"
	isThroughProxyKey = "isThroughProxy"
	RequestIDKey      = "RequestID"
)

var cache = NewLRUCache(1024 * 1024 * 1024) // 1GB

func main() {
	engine := gin.New()
	// 使用自定义的日志记录器中间件
	engine.Use(CustomLogger(os.Stdout), gin.Recovery(), RequestIDMiddleware())
	// 反向代理
	proxy := NewProxy("http://seer2.61.com", cache, 15*24*time.Hour)
	engine.Use(func(c *gin.Context) {
		host := c.Request.Host
		if host == "api.ex.com" {
			// 执行api路由逻辑
			c.Next()
		} else {
			c.Set(isThroughProxyKey, true)
			// 检查本地是否存在对应的静态文件
			filePath := filepath.Join("wwwroot", c.Request.URL.Path) // 静态文件路径，可根据需要进行调整
			if len(filePath) == 0 {
				proxy.ReverseProxy(c)
			}
			_, err := os.Stat(filePath)
			if err == nil {
				c.File(filePath)
				c.Set(sourceFlagKey, modified)
				c.Writer.Header().Set(sourceFlagKey, modified)
			} else {
				// 若静态文件不存在，使用反向代理
				proxy.ReverseProxy(c)
				c.Set(sourceFlagKey, original)
				c.Writer.Header().Set(sourceFlagKey, original)
			}
			// 若静态文件存在则直接返回
			// 允许缓存但强制客户端每次重新查询缓存是否过期
			// 我们可以接受再返回304，但不能接受在缓存期内就根本不发起任何请求。这将使我们的修改在缓存期内失效。
			c.Writer.Header().Set("Cache-control", "max-age=31536000, must-revalidate")
		}
	})
	engine.Run(":80")
}

// RequestIDMiddleware 是一个 Gin 中间件，用于为每个请求分配一个全局唯一的请求 ID
func RequestIDMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 生成一个全局唯一的请求 ID
		requestID := uuid.New().String()

		// 将请求 ID 存储在 Gin 上下文中，以便后续处理程序使用
		c.Set(RequestIDKey, requestID)

		// 继续处理请求
		c.Next()

		// 设置响应头中的请求 ID
		c.Writer.Header().Set("X-Request-ID", requestID)
	}
}

// 登录用户
func loginUser(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"message": "User login",
	})
}

// 注册用户
func registerUser(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"message": "User registration",
	})
}
