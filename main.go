package main

import (
	"context"
	"fmt"
	"os"

	"github.com/TIANLI0/LayerKit/config"
	"github.com/TIANLI0/LayerKit/handler"
	"github.com/TIANLI0/LayerKit/middleware"
	"github.com/TIANLI0/LayerKit/service"
	"github.com/TIANLI0/LayerKit/utils"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

var (
	Version   = "dev"
	BuildTime = "unknown"
	BuildID   = "unknown"
	GitCommit = "unknown"
	GitBranch = "unknown"
)

func main() {
	// 加载配置
	cfg := config.New()

	// 初始化日志
	if err := utils.InitLogger(cfg.Server.Mode); err != nil {
		fmt.Printf("Failed to initialize logger: %v\n", err)
		os.Exit(1)
	}
	defer utils.Sync()

	utils.Logger.Info("starting LayerKit server",
		zap.String("version", Version),
		zap.String("build_time", BuildTime),
		zap.String("git_commit", GitCommit),
		zap.String("git_branch", GitBranch))

	// 确保上传目录存在
	if err := os.MkdirAll(cfg.Upload.UploadDir, 0755); err != nil {
		utils.Logger.Fatal("failed to create upload directory", zap.Error(err))
	}

	// 初始化Redis
	redisService := service.NewRedisService(&cfg.Redis)
	ctx := context.Background()
	if err := redisService.Ping(ctx); err != nil {
		utils.Logger.Warn("redis connection failed, cache disabled", zap.Error(err))
	} else {
		utils.Logger.Info("redis connected successfully")
	}
	defer redisService.Close()

	// 初始化GrabCut服务
	grabCutService := service.NewGrabCutService(&cfg.GrabCut)

	// 初始化Handler
	uploadHandler := handler.NewUploadHandler(cfg, redisService, grabCutService)

	// 设置Gin模式
	gin.SetMode(cfg.Server.Mode)

	// 创建路由
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(middleware.Logger())
	r.Use(middleware.CORS())

	// 静态文件服务
	r.Static("/static", "./static")
	r.StaticFile("/", "./static/index.html")

	// 健康检查和版本信息
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status":  "ok",
			"version": Version,
		})
	})

	r.GET("/version", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"version":    Version,
			"build_time": BuildTime,
			"build_id":   BuildID,
			"git_commit": GitCommit,
			"git_branch": GitBranch,
		})
	})

	// API路由
	api := r.Group("/api/v1")
	{
		api.POST("/upload", uploadHandler.Upload)
		api.GET("/layer/:md5", uploadHandler.GetByMD5)
	}

	// 启动服务器
	utils.Logger.Info("server starting", zap.String("port", cfg.Server.Port))
	if err := r.Run(cfg.Server.Port); err != nil {
		utils.Logger.Fatal("failed to start server", zap.Error(err))
	}
}
