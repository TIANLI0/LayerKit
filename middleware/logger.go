package middleware

import (
	"time"

	"github.com/TIANLI0/LayerKit/utils"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// Logger Zap日志中间件
func Logger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery

		c.Next()

		cost := time.Since(start)

		utils.Logger.Info("request",
			zap.String("method", c.Request.Method),
			zap.String("path", path),
			zap.String("query", query),
			zap.Int("status", c.Writer.Status()),
			zap.String("ip", c.ClientIP()),
			zap.Duration("cost", cost),
			zap.String("user_agent", c.Request.UserAgent()),
		)
	}
}
