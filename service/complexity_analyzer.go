package service

import (
	"gocv.io/x/gocv"
)

// ComplexityAnalyzer 负责分析图像的复杂度
type ComplexityAnalyzer struct {
	portraitDetector *PortraitDetector
}

type ComplexityInfo struct {
	Level         string
	EdgeDensity   float64
	ColorVariance float64
	IsPortrait    bool
}

// NewComplexityAnalyzer 创建一个新的ComplexityAnalyzer实例
func NewComplexityAnalyzer() *ComplexityAnalyzer {
	return &ComplexityAnalyzer{
		portraitDetector: NewPortraitDetector(),
	}
}

// Analyze 分析图像的复杂度
func (ca *ComplexityAnalyzer) Analyze(img *gocv.Mat) ComplexityInfo {
	edgeDensity := ca.calculateEdgeDensity(img)
	colorVariance := ca.calculateColorVariance(img)
	isPortrait := ca.portraitDetector.IsPortrait(img)

	var level string
	if isPortrait {
		level = "portrait"
	} else if edgeDensity < 0.05 && colorVariance < 30 {
		level = "simple"
	} else if edgeDensity > 0.15 || colorVariance > 60 {
		level = "complex"
	} else {
		level = "medium"
	}

	return ComplexityInfo{
		Level:         level,
		EdgeDensity:   edgeDensity,
		ColorVariance: colorVariance,
		IsPortrait:    isPortrait,
	}
}

// calculateEdgeDensity 计算图像的边缘密度
func (ca *ComplexityAnalyzer) calculateEdgeDensity(img *gocv.Mat) float64 {
	gray := gocv.NewMat()
	defer gray.Close()
	gocv.CvtColor(*img, &gray, gocv.ColorBGRToGray)

	edges := gocv.NewMat()
	defer edges.Close()
	gocv.Canny(gray, &edges, 50, 150)

	edgePixels := float64(gocv.CountNonZero(edges))
	totalPixels := float64(img.Rows() * img.Cols())

	return edgePixels / totalPixels
}

// calculateColorVariance 计算图像的颜色方差
func (ca *ComplexityAnalyzer) calculateColorVariance(img *gocv.Mat) float64 {
	lab := gocv.NewMat()
	defer lab.Close()
	gocv.CvtColor(*img, &lab, gocv.ColorBGRToLab)

	mean := gocv.NewMat()
	stddev := gocv.NewMat()
	defer mean.Close()
	defer stddev.Close()
	gocv.MeanStdDev(lab, &mean, &stddev)

	variance := 0.0
	for i := 0; i < stddev.Rows(); i++ {
		variance += stddev.GetDoubleAt(i, 0)
	}

	return variance / float64(stddev.Rows())
}
