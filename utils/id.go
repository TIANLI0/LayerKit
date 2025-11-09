package utils

import (
	"time"
)

// GenerateID 生成基于时间戳的ID
func GenerateID() int64 {
	return time.Now().UnixNano()
}
