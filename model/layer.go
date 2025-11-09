package model

// LayerResult 分层结果
type LayerResult struct {
	MD5       string  `json:"md5"`
	Width     int     `json:"width"`
	Height    int     `json:"height"`
	Layers    []Layer `json:"layers"`
	Timestamp int64   `json:"timestamp"`
}

// Layer 单个图层信息
type Layer struct {
	ID          int     `json:"id"`
	Type        string  `json:"type"` // foreground, background
	BoundingBox BBox    `json:"bounding_box"`
	Mask        string  `json:"mask"` // base64编码的mask数据
	Confidence  float64 `json:"confidence"`
}

// BBox 边界框
type BBox struct {
	X      int `json:"x"`
	Y      int `json:"y"`
	Width  int `json:"width"`
	Height int `json:"height"`
}

// UploadResponse 上传响应
type UploadResponse struct {
	Success bool         `json:"success"`
	Message string       `json:"message"`
	Data    *LayerResult `json:"data,omitempty"`
}

// ErrorResponse 错误响应
type ErrorResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Error   string `json:"error,omitempty"`
}
