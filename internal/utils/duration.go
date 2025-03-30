package utils

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// 持续时间部分的正则表达式
var (
	durationRegex = regexp.MustCompile(`^P(?:(\d+)Y)?(?:(\d+)M)?(?:(\d+)D)?(?:T(?:(\d+)H)?(?:(\d+)M)?(?:(\d+(?:\.\d+)?)S)?)?$`)
	weekRegex     = regexp.MustCompile(`^P(\d+)W$`)
)

// ParseISO8601Duration 解析ISO 8601格式的持续时间字符串
// 支持格式如: P1Y2M3DT4H5M6S, PT1H, P1D, P1W等
// 如果无法解析，则返回错误
func ParseISO8601Duration(durationStr string) (time.Duration, error) {
	// 处理空字符串
	if durationStr == "" {
		return 0, fmt.Errorf("空的持续时间字符串")
	}

	// 处理周格式 (PnW)
	if match := weekRegex.FindStringSubmatch(durationStr); match != nil {
		weeks, _ := strconv.Atoi(match[1])
		return time.Duration(weeks) * 7 * 24 * time.Hour, nil
	}

	// 处理标准格式 (PnYnMnDTnHnMnS)
	match := durationRegex.FindStringSubmatch(durationStr)
	if match == nil {
		return 0, fmt.Errorf("无效的ISO 8601持续时间格式: %s", durationStr)
	}

	// 解析各个部分
	var duration time.Duration

	// 年
	if match[1] != "" {
		years, _ := strconv.Atoi(match[1])
		duration += time.Duration(years) * 365 * 24 * time.Hour
	}

	// 月
	if match[2] != "" {
		months, _ := strconv.Atoi(match[2])
		duration += time.Duration(months) * 30 * 24 * time.Hour
	}

	// 日
	if match[3] != "" {
		days, _ := strconv.Atoi(match[3])
		duration += time.Duration(days) * 24 * time.Hour
	}

	// 时
	if match[4] != "" {
		hours, _ := strconv.Atoi(match[4])
		duration += time.Duration(hours) * time.Hour
	}

	// 分
	if match[5] != "" {
		minutes, _ := strconv.Atoi(match[5])
		duration += time.Duration(minutes) * time.Minute
	}

	// 秒
	if match[6] != "" {
		seconds, _ := strconv.ParseFloat(match[6], 64)
		duration += time.Duration(seconds * float64(time.Second))
	}

	return duration, nil
}

// FormatDuration 将time.Duration格式化为ISO 8601持续时间格式
func FormatDuration(d time.Duration) string {
	var parts []string

	// 提取各个时间单位
	days := int(d.Hours()) / 24
	hours := int(d.Hours()) % 24
	minutes := int(d.Minutes()) % 60
	seconds := int(d.Seconds()) % 60
	milliseconds := int(d.Milliseconds()) % 1000

	// 构建ISO 8601格式
	if days > 0 {
		parts = append(parts, fmt.Sprintf("%dD", days))
	}

	var timeParts []string
	if hours > 0 {
		timeParts = append(timeParts, fmt.Sprintf("%dH", hours))
	}
	if minutes > 0 {
		timeParts = append(timeParts, fmt.Sprintf("%dM", minutes))
	}
	if seconds > 0 || milliseconds > 0 {
		if milliseconds > 0 {
			timeParts = append(timeParts, fmt.Sprintf("%d.%03dS", seconds, milliseconds))
		} else {
			timeParts = append(timeParts, fmt.Sprintf("%dS", seconds))
		}
	}

	// 组合结果
	result := "P"
	if len(parts) > 0 {
		result += strings.Join(parts, "")
	}

	if len(timeParts) > 0 {
		result += "T" + strings.Join(timeParts, "")
	} else if len(parts) == 0 {
		// 如果没有天和时间部分，至少添加0秒
		result += "T0S"
	}

	return result
}

// GetDurationOrDefault 从字符串解析持续时间，如果解析失败则返回默认值（秒）
func GetDurationOrDefault(durationStr string, defaultSeconds int) time.Duration {
	if durationStr == "" {
		return time.Duration(defaultSeconds) * time.Second
	}

	// 尝试解析ISO 8601格式
	duration, err := ParseISO8601Duration(durationStr)
	if err == nil {
		return duration
	}

	// 尝试解析Go的持续时间格式 (如: "1h30m")
	duration, err = time.ParseDuration(durationStr)
	if err == nil {
		return duration
	}

	// 最后尝试解析为秒
	if seconds, err := strconv.Atoi(durationStr); err == nil {
		return time.Duration(seconds) * time.Second
	}

	// 返回默认值
	return time.Duration(defaultSeconds) * time.Second
}
