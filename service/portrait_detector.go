package service

import (
	"image"

	"gocv.io/x/gocv"
)

// PortraitDetector 负责检测图像中的人像特征
type PortraitDetector struct{}

func NewPortraitDetector() *PortraitDetector {
	return &PortraitDetector{}
}

// DetectSkin 检测图像中的皮肤区域
func (pd *PortraitDetector) DetectSkin(img *gocv.Mat) gocv.Mat {
	ycrcb := gocv.NewMat()
	defer ycrcb.Close()
	gocv.CvtColor(*img, &ycrcb, gocv.ColorBGRToYCrCb)

	lower := gocv.Scalar{Val1: 0, Val2: 133, Val3: 77, Val4: 0}
	upper := gocv.Scalar{Val1: 255, Val2: 173, Val3: 127, Val4: 255}

	skinMask := gocv.NewMat()
	gocv.InRangeWithScalar(ycrcb, lower, upper, &skinMask)

	kernel := gocv.GetStructuringElement(gocv.MorphEllipse, image.Point{X: 5, Y: 5})
	defer kernel.Close()

	gocv.MorphologyEx(skinMask, &skinMask, gocv.MorphClose, kernel)
	gocv.MorphologyEx(skinMask, &skinMask, gocv.MorphOpen, kernel)

	return skinMask
}

// DetectFace 检测图像中的人脸位置
func (pd *PortraitDetector) DetectFace(img *gocv.Mat) []image.Rectangle {
	gray := gocv.NewMat()
	defer gray.Close()
	gocv.CvtColor(*img, &gray, gocv.ColorBGRToGray)

	classifier := gocv.NewCascadeClassifier()
	defer classifier.Close()

	if !classifier.Load("haarcascade_frontalface_default.xml") {
		return nil
	}

	return classifier.DetectMultiScale(gray)
}

func (pd *PortraitDetector) IsPortrait(img *gocv.Mat) bool {
	skinMask := pd.DetectSkin(img)
	defer skinMask.Close()

	totalPixels := float64(img.Rows() * img.Cols())
	skinPixels := float64(gocv.CountNonZero(skinMask))
	skinRatio := skinPixels / totalPixels

	return skinRatio > 0.15
}

// EnhancePortraitMask 使用皮肤检测结果增强原始人像掩码
func (pd *PortraitDetector) EnhancePortraitMask(originalMask, img *gocv.Mat) gocv.Mat {
	skinMask := pd.DetectSkin(img)
	defer skinMask.Close()

	kernel := gocv.GetStructuringElement(gocv.MorphEllipse, image.Point{X: 15, Y: 15})
	defer kernel.Close()

	dilatedSkin := gocv.NewMat()
	defer dilatedSkin.Close()
	gocv.Dilate(skinMask, &dilatedSkin, kernel)

	enhanced := gocv.NewMat()
	gocv.BitwiseOr(*originalMask, dilatedSkin, &enhanced)

	return enhanced
}
