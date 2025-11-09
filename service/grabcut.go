package service

import (
	"context"
	"encoding/base64"
	"fmt"
	"image"
	"time"

	"github.com/TIANLI0/LayerKit/config"
	"github.com/TIANLI0/LayerKit/model"
	"github.com/TIANLI0/LayerKit/utils"
	"go.uber.org/zap"
	"gocv.io/x/gocv"
)

// GrabCutService 负责图像分层处理
type GrabCutService struct {
	iterations         int
	borderSize         int
	semaphore          chan struct{}
	queueTimeout       time.Duration
	cleanupTempFiles   bool
	complexityAnalyzer *ComplexityAnalyzer
	saliencyDetector   *SaliencyDetector
	maskProcessor      *MaskProcessor
	portraitDetector   *PortraitDetector
}

func NewGrabCutService(cfg *config.GrabCutConfig) *GrabCutService {
	return &GrabCutService{
		iterations:         cfg.Iterations,
		borderSize:         cfg.BorderSize,
		semaphore:          make(chan struct{}, cfg.MaxConcurrent),
		queueTimeout:       time.Duration(cfg.QueueTimeout) * time.Second,
		cleanupTempFiles:   cfg.CleanupTempFiles,
		complexityAnalyzer: NewComplexityAnalyzer(),
		saliencyDetector:   NewSaliencyDetector(),
		maskProcessor:      NewMaskProcessor(),
		portraitDetector:   NewPortraitDetector(),
	}
}

// ProcessImage 处理图片并返回分层结果
func (s *GrabCutService) ProcessImage(imagePath string, md5 string, maxForegroundOnly bool) (*model.LayerResult, error) {
	// 并发控制
	ctx, cancel := context.WithTimeout(context.Background(), s.queueTimeout)
	defer cancel()

	select {
	case s.semaphore <- struct{}{}:
		defer func() { <-s.semaphore }()
	case <-ctx.Done():
		return nil, fmt.Errorf("处理队列已满，请稍后重试")
	}

	startTime := time.Now()

	// 读取图片
	img := gocv.IMRead(imagePath, gocv.IMReadColor)
	if img.Empty() {
		return nil, fmt.Errorf("failed to read image")
	}
	defer img.Close()

	width := img.Cols()
	height := img.Rows()

	utils.Logger.Info("processing image",
		zap.String("md5", md5),
		zap.Int("width", width),
		zap.Int("height", height))

	// 智能缩放
	scaledImg, scale := s.smartResize(&img, 1200)
	defer scaledImg.Close()

	scaledWidth := scaledImg.Cols()
	scaledHeight := scaledImg.Rows()

	complexity := s.complexityAnalyzer.Analyze(&scaledImg)
	utils.Logger.Info("scene analyzed",
		zap.String("level", complexity.Level),
		zap.Bool("is_portrait", complexity.IsPortrait))

	var initRect image.Rectangle
	var mask gocv.Mat

	if complexity.Level == "simple" {
		border := s.borderSize
		if border < 10 {
			border = int(float64(scaledWidth) * 0.05)
		}
		initRect = image.Rect(border, border, scaledWidth-border, scaledHeight-border)
		mask = gocv.NewMat()
	} else {
		saliencyMap := s.saliencyDetector.Detect(&scaledImg)
		defer saliencyMap.Close()

		initRect = s.saliencyDetector.ExtractRect(&saliencyMap, scaledWidth, scaledHeight)
		mask = s.saliencyDetector.CreateMask(&saliencyMap, scaledWidth, scaledHeight)
	}
	defer mask.Close()

	bgdModel := gocv.NewMat()
	defer bgdModel.Close()
	fgdModel := gocv.NewMat()
	defer fgdModel.Close()

	iterations := s.iterations
	switch complexity.Level {
	case "simple":
		iterations = max(3, s.iterations-2)
	case "portrait":
		iterations = s.iterations + 1
	case "complex":
		iterations = s.iterations + 2
	}

	if mask.Empty() {
		gocv.GrabCut(scaledImg, &mask, initRect, &bgdModel, &fgdModel, iterations, gocv.GCInitWithRect)
	} else {
		gocv.GrabCut(scaledImg, &mask, image.Rectangle{}, &bgdModel, &fgdModel, iterations, gocv.GCInitWithMask)
	}

	if complexity.Level != "simple" {
		gocv.GrabCut(scaledImg, &mask, image.Rectangle{}, &bgdModel, &fgdModel, 2, gocv.GCInitWithMask)
	}

	fgMask := s.maskProcessor.ExtractForeground(&mask)
	defer fgMask.Close()

	if complexity.IsPortrait {
		enhanced := s.portraitDetector.EnhancePortraitMask(&fgMask, &scaledImg)
		fgMask.Close()
		fgMask = enhanced

		detailRefined := s.maskProcessor.DetailPreservingRefine(&fgMask, &scaledImg)
		fgMask.Close()
		fgMask = detailRefined
	}

	kernelSize := 3
	if complexity.Level == "complex" || complexity.Level == "portrait" {
		kernelSize = 5
	}
	optimized := s.maskProcessor.MorphologyOptimize(&fgMask, kernelSize)
	fgMask.Close()
	fgMask = optimized

	if complexity.Level != "simple" {
		refined := s.maskProcessor.RefineEdges(&fgMask)
		fgMask.Close()
		fgMask = refined
	}

	// 还原到原始尺寸
	if scale != 1.0 {
		resizedMask := gocv.NewMat()
		gocv.Resize(fgMask, &resizedMask, image.Point{X: width, Y: height}, 0, 0, gocv.InterpolationLinear)
		gocv.Threshold(resizedMask, &resizedMask, 127, 255, gocv.ThresholdBinary)
		fgMask.Close()
		fgMask = resizedMask
	}

	if maxForegroundOnly {
		largest := s.maskProcessor.KeepLargest(&fgMask)
		fgMask.Close()
		fgMask = largest
	}
	fgBBox := s.calculateBoundingBox(&fgMask)
	fgMaskBase64 := s.encodeMask(&fgMask)

	bgMask := gocv.NewMat()
	defer bgMask.Close()
	gocv.BitwiseNot(fgMask, &bgMask)
	bgMaskBase64 := s.encodeMask(&bgMask)

	fgConfidence := s.calculateConfidence(&fgMask, width, height)

	result := &model.LayerResult{
		MD5:       md5,
		Width:     width,
		Height:    height,
		Timestamp: time.Now().Unix(),
		Layers: []model.Layer{
			{
				ID:          1,
				Type:        "foreground",
				BoundingBox: fgBBox,
				Mask:        fgMaskBase64,
				Confidence:  fgConfidence,
			},
			{
				ID:          2,
				Type:        "background",
				BoundingBox: model.BBox{X: 0, Y: 0, Width: width, Height: height},
				Mask:        bgMaskBase64,
				Confidence:  1.0 - fgConfidence,
			},
		},
	}

	utils.Logger.Info("image processed successfully",
		zap.String("md5", md5),
		zap.Duration("duration", time.Since(startTime)),
		zap.Float64("foreground_confidence", fgConfidence),
		zap.String("complexity", complexity.Level))

	return result, nil
}

// calculateBoundingBox 计算掩码的边界框
func (s *GrabCutService) calculateBoundingBox(mask *gocv.Mat) model.BBox {
	contours := gocv.FindContours(*mask, gocv.RetrievalExternal, gocv.ChainApproxSimple)

	if contours.Size() == 0 {
		return model.BBox{}
	}

	var union image.Rectangle
	for i := 0; i < contours.Size(); i++ {
		c := contours.At(i)
		r := gocv.BoundingRect(c)
		if i == 0 {
			union = r
		} else {
			union = union.Union(r)
		}
	}

	return model.BBox{
		X:      union.Min.X,
		Y:      union.Min.Y,
		Width:  union.Dx(),
		Height: union.Dy(),
	}
}

// encodeMask 将掩码编码为Base64字符串
func (s *GrabCutService) encodeMask(mask *gocv.Mat) string {
	data, err := gocv.IMEncode(".png", *mask)
	if err != nil {
		utils.Logger.Error("failed to encode mask", zap.Error(err))
		return ""
	}
	defer data.Close()

	return base64.StdEncoding.EncodeToString(data.GetBytes())
}

// calculateConfidence 计算前景掩码的置信度
func (s *GrabCutService) calculateConfidence(mask *gocv.Mat, width, height int) float64 {
	confidence := float64(gocv.CountNonZero(*mask)) / float64(width*height)
	if confidence < 0.05 {
		confidence = 0.05
	}
	if confidence > 0.95 {
		confidence = 0.95
	}
	return confidence
}

// smartResize 智能缩放图像以适应最大尺寸
func (s *GrabCutService) smartResize(img *gocv.Mat, maxSize int) (gocv.Mat, float64) {
	width := img.Cols()
	height := img.Rows()
	maxDim := max(width, height)
	if maxDim <= maxSize {
		return img.Clone(), 1.0
	}

	scale := float64(maxSize) / float64(maxDim)
	newWidth := int(float64(width) * scale)
	newHeight := int(float64(height) * scale)

	resized := gocv.NewMat()
	gocv.Resize(*img, &resized, image.Point{X: newWidth, Y: newHeight}, 0, 0, gocv.InterpolationArea)

	return resized, scale
}
