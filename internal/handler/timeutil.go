package handler

import "time"

// parseTimeRFC3339 解析 RFC3339 格式时间字符串。
func parseTimeRFC3339(s string) (time.Time, error) {
	return time.Parse(time.RFC3339, s)
}

// timeNow 返回当前时间，便于测试替换。
var timeNow = time.Now
