package handler

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/TIANLI0/LayerKit/config"
	"github.com/TIANLI0/LayerKit/model"
	"github.com/TIANLI0/LayerKit/service"
	"github.com/TIANLI0/LayerKit/utils"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type UploadHandler struct {
	cfg            *config.Config
	redisService   *service.RedisService
	grabCutService *service.GrabCutService
}

func NewUploadHandler(cfg *config.Config, redis *service.RedisService, grabCut *service.GrabCutService) *UploadHandler {
	return &UploadHandler{
		cfg:            cfg,
		redisService:   redis,
		grabCutService: grabCut,
	}
}

// Upload 处理图片上传
func (h *UploadHandler) Upload(c *gin.Context) {
	file, err := c.FormFile("image")
	if err != nil {
		utils.Logger.Error("failed to get uploaded file", zap.Error(err))
		c.JSON(http.StatusBadRequest, model.ErrorResponse{
			Success: false,
			Message: "请上传图片文件",
			Error:   err.Error(),
		})
		return
	}

	// 验证文件大小
	if file.Size > h.cfg.Upload.MaxSize {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{
			Success: false,
			Message: fmt.Sprintf("文件大小超过限制 (%d MB)", h.cfg.Upload.MaxSize/(1024*1024)),
		})
		return
	}

	// 验证文件类型
	contentType := file.Header.Get("Content-Type")
	if !h.isAllowedType(contentType) {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{
			Success: false,
			Message: "不支持的文件类型，仅支持 JPEG/PNG",
		})
		return
	}

	// 生成文件名
	ext := filepath.Ext(file.Filename)
	filename := fmt.Sprintf("%d%s", utils.GenerateID(), ext)
	savePath := filepath.Join(h.cfg.Upload.UploadDir, filename)

	// 保存文件
	if err := c.SaveUploadedFile(file, savePath); err != nil {
		utils.Logger.Error("failed to save file", zap.Error(err))
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{
			Success: false,
			Message: "保存文件失败",
			Error:   err.Error(),
		})
		return
	}

	// 计算MD5
	md5, err := utils.FileMD5(savePath)
	if err != nil {
		utils.Logger.Error("failed to calculate md5", zap.Error(err))
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{
			Success: false,
			Message: "计算文件哈希失败",
			Error:   err.Error(),
		})
		return
	}

	// 确保文件在处理完成后被删除（如果配置启用）
	if h.cfg.GrabCut.CleanupTempFiles {
		defer func() {
			if err := os.Remove(savePath); err != nil {
				utils.Logger.Warn("failed to delete temp file",
					zap.String("file", savePath),
					zap.Error(err))
			} else {
				utils.Logger.Debug("temp file deleted",
					zap.String("file", savePath))
			}
		}()
	}

	// 获取参数
	maxForegroundOnly := c.DefaultPostForm("max_foreground_only", "false") == "true"

	utils.Logger.Info("file uploaded",
		zap.String("filename", filename),
		zap.String("md5", md5),
		zap.Int64("size", file.Size),
		zap.Bool("max_foreground_only", maxForegroundOnly))

	// 检查缓存（带参数区分）
	ctx := context.Background()
	cacheKey := md5
	if maxForegroundOnly {
		cacheKey = md5 + ":max_fg"
	}

	cachedResult, err := h.redisService.GetLayerResult(ctx, cacheKey)
	if err != nil {
		utils.Logger.Warn("failed to get cache", zap.Error(err))
	}

	if cachedResult != nil {
		utils.Logger.Info("cache hit", zap.String("cache_key", cacheKey))
		c.JSON(http.StatusOK, model.UploadResponse{
			Success: true,
			Message: "处理成功（来自缓存）",
			Data:    cachedResult,
		})
		return
	}

	// 处理图片
	result, err := h.grabCutService.ProcessImage(savePath, md5, maxForegroundOnly)
	if err != nil {
		utils.Logger.Error("failed to process image", zap.Error(err))
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{
			Success: false,
			Message: "图片处理失败",
			Error:   err.Error(),
		})
		return
	}

	// 保存到缓存
	if err := h.redisService.SetLayerResult(ctx, cacheKey, result); err != nil {
		utils.Logger.Warn("failed to set cache", zap.Error(err))
	}

	c.JSON(http.StatusOK, model.UploadResponse{
		Success: true,
		Message: "处理成功",
		Data:    result,
	})
}

// GetByMD5 根据MD5获取分层信息
func (h *UploadHandler) GetByMD5(c *gin.Context) {
	md5 := c.Param("md5")
	if md5 == "" {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{
			Success: false,
			Message: "MD5参数缺失",
		})
		return
	}

	ctx := context.Background()
	result, err := h.redisService.GetLayerResult(ctx, md5)
	if err != nil {
		utils.Logger.Error("failed to get layer result", zap.Error(err))
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{
			Success: false,
			Message: "查询失败",
			Error:   err.Error(),
		})
		return
	}

	if result == nil {
		c.JSON(http.StatusNotFound, model.ErrorResponse{
			Success: false,
			Message: "未找到该图片的分层信息",
		})
		return
	}

	c.JSON(http.StatusOK, model.UploadResponse{
		Success: true,
		Message: "查询成功",
		Data:    result,
	})
}

func (h *UploadHandler) isAllowedType(contentType string) bool {
	for _, allowed := range h.cfg.Upload.AllowedTypes {
		if strings.EqualFold(contentType, allowed) {
			return true
		}
	}
	return false
}
