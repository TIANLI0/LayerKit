package service

import (
	"image"

	"gocv.io/x/gocv"
)

// SaliencyDetector 负责检测图像的显著性区域
type SaliencyDetector struct{}

func NewSaliencyDetector() *SaliencyDetector {
	return &SaliencyDetector{}
}

// Detect 计算图像的显著性图
func (sd *SaliencyDetector) Detect(img *gocv.Mat) gocv.Mat {
	gray := gocv.NewMat()
	defer gray.Close()
	gocv.CvtColor(*img, &gray, gocv.ColorBGRToGray)

	gradX := gocv.NewMat()
	gradY := gocv.NewMat()
	defer gradX.Close()
	defer gradY.Close()

	gocv.Sobel(gray, &gradX, gocv.MatTypeCV16S, 1, 0, 3, 1, 0, gocv.BorderDefault)
	gocv.Sobel(gray, &gradY, gocv.MatTypeCV16S, 0, 1, 3, 1, 0, gocv.BorderDefault)

	absGradX := gocv.NewMat()
	absGradY := gocv.NewMat()
	defer absGradX.Close()
	defer absGradY.Close()

	gocv.ConvertScaleAbs(gradX, &absGradX, 1, 0)
	gocv.ConvertScaleAbs(gradY, &absGradY, 1, 0)

	gradient := gocv.NewMat()
	defer gradient.Close()
	gocv.AddWeighted(absGradX, 0.5, absGradY, 0.5, 0, &gradient)

	blurred := gocv.NewMat()
	defer blurred.Close()
	gocv.GaussianBlur(gradient, &blurred, image.Point{X: 21, Y: 21}, 0, 0, gocv.BorderDefault)

	saliency := gocv.NewMat()
	gocv.Threshold(blurred, &saliency, 0, 255, gocv.ThresholdOtsu)

	return saliency
}

// ExtractRect 提取显著性区域的边界矩形
func (sd *SaliencyDetector) ExtractRect(saliency *gocv.Mat, width, height int) image.Rectangle {
	kernel := gocv.GetStructuringElement(gocv.MorphEllipse, image.Point{X: 21, Y: 21})
	defer kernel.Close()

	dilated := gocv.NewMat()
	defer dilated.Close()
	gocv.Dilate(*saliency, &dilated, kernel)

	contours := gocv.FindContours(dilated, gocv.RetrievalExternal, gocv.ChainApproxSimple)

	if contours.Size() == 0 {
		border := int(float64(width) * 0.1)
		return image.Rect(border, border, width-border, height-border)
	}

	var maxRect image.Rectangle
	maxArea := 0.0

	for i := 0; i < contours.Size(); i++ {
		area := gocv.ContourArea(contours.At(i))
		if area > maxArea {
			maxArea = area
			maxRect = gocv.BoundingRect(contours.At(i))
		}
	}

	padding := int(float64(maxRect.Dx()) * 0.05)
	maxRect.Min.X = max(0, maxRect.Min.X-padding)
	maxRect.Min.Y = max(0, maxRect.Min.Y-padding)
	maxRect.Max.X = min(width, maxRect.Max.X+padding)
	maxRect.Max.Y = min(height, maxRect.Max.Y+padding)

	return maxRect
}

// CreateMask 根据显著性图创建GrabCut掩码
func (sd *SaliencyDetector) CreateMask(saliency *gocv.Mat, width, height int) gocv.Mat {
	mask := gocv.NewMatWithSize(height, width, gocv.MatTypeCV8U)
	mask.SetTo(gocv.NewScalar(2, 0, 0, 0))

	borderSize := int(float64(width) * 0.03)
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			if x < borderSize || x >= width-borderSize ||
				y < borderSize || y >= height-borderSize {
				mask.SetUCharAt(y, x, 0)
			}
		}
	}

	kernel := gocv.GetStructuringElement(gocv.MorphEllipse, image.Point{X: 11, Y: 11})
	defer kernel.Close()

	dilated := gocv.NewMat()
	defer dilated.Close()
	gocv.Dilate(*saliency, &dilated, kernel)

	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			if dilated.GetUCharAt(y, x) > 128 {
				mask.SetUCharAt(y, x, 3)
			}
		}
	}

	return mask
}
