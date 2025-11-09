package service

import (
	"image"
	"image/color"

	"gocv.io/x/gocv"
)

// MaskProcessor 负责处理图像掩码
type MaskProcessor struct{}

func NewMaskProcessor() *MaskProcessor {
	return &MaskProcessor{}
}

// ExtractForeground 提取前景掩码
func (mp *MaskProcessor) ExtractForeground(mask *gocv.Mat) gocv.Mat {
	fgMask := gocv.NewMat()
	tmp1 := gocv.NewMatFromScalar(gocv.Scalar{Val1: 1}, gocv.MatTypeCV8U)
	defer tmp1.Close()
	gocv.Compare(*mask, tmp1, &fgMask, gocv.CompareEQ)

	fgMaskPr := gocv.NewMat()
	defer fgMaskPr.Close()
	tmp2 := gocv.NewMatFromScalar(gocv.Scalar{Val1: 3}, gocv.MatTypeCV8U)
	defer tmp2.Close()
	gocv.Compare(*mask, tmp2, &fgMaskPr, gocv.CompareEQ)

	combined := gocv.NewMat()
	gocv.BitwiseOr(fgMask, fgMaskPr, &combined)
	fgMask.Close()

	return combined
}

// MorphologyOptimize 优化掩码的形态学结构
func (mp *MaskProcessor) MorphologyOptimize(mask *gocv.Mat, kernelSize int) gocv.Mat {
	kernel := gocv.GetStructuringElement(gocv.MorphEllipse, image.Point{X: kernelSize, Y: kernelSize})
	defer kernel.Close()

	opened := gocv.NewMat()
	gocv.MorphologyEx(*mask, &opened, gocv.MorphOpen, kernel)

	closed := gocv.NewMat()
	gocv.MorphologyEx(opened, &closed, gocv.MorphClose, kernel)
	opened.Close()

	return closed
}

// RefineEdges 精细化掩码边缘
func (mp *MaskProcessor) RefineEdges(mask *gocv.Mat) gocv.Mat {
	kernel := gocv.GetStructuringElement(gocv.MorphEllipse, image.Point{X: 2, Y: 2})
	defer kernel.Close()

	refined := gocv.NewMat()
	gocv.Dilate(*mask, &refined, kernel)

	blurred := gocv.NewMat()
	gocv.GaussianBlur(refined, &blurred, image.Point{X: 3, Y: 3}, 0, 0, gocv.BorderDefault)
	refined.Close()

	final := gocv.NewMat()
	gocv.Threshold(blurred, &final, 127, 255, gocv.ThresholdBinary)
	blurred.Close()

	return final
}

// KeepLargest 保留掩码中最大的连通区域
func (mp *MaskProcessor) KeepLargest(mask *gocv.Mat) gocv.Mat {
	contours := gocv.FindContours(*mask, gocv.RetrievalExternal, gocv.ChainApproxSimple)

	if contours.Size() == 0 {
		return *mask
	}

	maxArea := 0.0
	maxIndex := 0
	for i := 0; i < contours.Size(); i++ {
		area := gocv.ContourArea(contours.At(i))
		if area > maxArea {
			maxArea = area
			maxIndex = i
		}
	}

	newMask := gocv.NewMatWithSize(mask.Rows(), mask.Cols(), gocv.MatTypeCV8U)
	white := color.RGBA{R: 255, G: 255, B: 255, A: 255}
	gocv.DrawContours(&newMask, contours, maxIndex, white, -1)

	return newMask
}

// DetailPreservingRefine 使用皮肤检测结果增强原始人像掩码
func (mp *MaskProcessor) DetailPreservingRefine(mask, img *gocv.Mat) gocv.Mat {
	gray := gocv.NewMat()
	defer gray.Close()
	gocv.CvtColor(*img, &gray, gocv.ColorBGRToGray)

	edges := gocv.NewMat()
	defer edges.Close()
	gocv.Canny(gray, &edges, 30, 90)

	kernel := gocv.GetStructuringElement(gocv.MorphRect, image.Point{X: 3, Y: 3})
	defer kernel.Close()

	dilatedEdges := gocv.NewMat()
	defer dilatedEdges.Close()
	gocv.Dilate(edges, &dilatedEdges, kernel)

	refined := mask.Clone()

	for y := 0; y < mask.Rows(); y++ {
		for x := 0; x < mask.Cols(); x++ {
			if dilatedEdges.GetUCharAt(y, x) > 0 {
				if mp.checkNeighborhood(mask, x, y) {
					refined.SetUCharAt(y, x, mask.GetUCharAt(y, x))
				}
			}
		}
	}

	return refined
}

// checkNeighborhood 检查邻域内的像素
func (mp *MaskProcessor) checkNeighborhood(mask *gocv.Mat, x, y int) bool {
	count := 0
	for dy := -2; dy <= 2; dy++ {
		for dx := -2; dx <= 2; dx++ {
			nx, ny := x+dx, y+dy
			if nx >= 0 && nx < mask.Cols() && ny >= 0 && ny < mask.Rows() {
				if mask.GetUCharAt(ny, nx) > 128 {
					count++
				}
			}
		}
	}
	return count > 12
}
