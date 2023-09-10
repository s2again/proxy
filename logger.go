package main

import (
	"fmt"
	"io"
	"time"

	"github.com/gin-gonic/gin"
)

// ANSI颜色转义序列常量
const (
	YellowTextOnBlack = "\033[33;40m"
	CyanTextOnWhite   = "\033[96;46m"
	BlueTextOnWhite   = "\033[34;107m"
	GreenTextOnBlack  = "\033[32;40m"
)

// CustomLogger 是自定义的日志记录器
func CustomLogger(output io.Writer, notlogged ...string) gin.HandlerFunc {
	return gin.LoggerWithConfig(gin.LoggerConfig{
		Formatter: func(param gin.LogFormatterParams) string {
			// 构造自定义的日志消息
			var statusColor, methodColor, resetColor string
			if param.IsOutputColor() {
				statusColor = param.StatusCodeColor()
				methodColor = param.MethodColor()
				resetColor = param.ResetColor()
			}

			if param.Latency > time.Minute {
				param.Latency = param.Latency.Truncate(time.Second)
			}
			var (
				requestCategory  string
				responseCategory string
			)
			if isThroughProxy, exist := param.Keys[isThroughProxyKey].(bool); exist && isThroughProxy {
				requestCategory = "proxy"
				sourceFlag, exist := param.Keys[sourceFlagKey].(string)
				if !exist {
					responseCategory = GreenTextOnBlack + "NotSetSourceFlag" + resetColor
				} else {
					switch sourceFlag {
					case modified:
						responseCategory = YellowTextOnBlack + sourceFlag + resetColor
					case original:
						responseCategory = CyanTextOnWhite + sourceFlag + resetColor
					default:
						responseCategory = BlueTextOnWhite + "UnknownSourceFlag" + resetColor
					}
				}
				responseCategory = fmt.Sprintf("%s", responseCategory)
			} else {
				requestCategory = "api"
			}
			requestID, exist := param.Keys[RequestIDKey]
			if !exist {
				responseCategory = "emptyRequestID"
			}
			return fmt.Sprintf("[GIN] %s-request %s| %s |%s %3d %s| %13v | %15s |%s %-7s %s %#v \nResponse as %s %d bytes. \n%s",
				requestCategory,
				requestID,
				param.TimeStamp.Format("2006/01/02 - 15:04:05.000000000"),
				statusColor, param.StatusCode, resetColor,
				param.Latency,
				param.ClientIP,
				methodColor, param.Method, resetColor,
				param.Path,
				responseCategory,
				param.BodySize,
				param.ErrorMessage,
			)
		},
		Output:    output,
		SkipPaths: notlogged,
	})
}
