package utils

import (
	"path/filepath"
	"runtime"
)

// JoinPath 连接路径
func JoinPath(paths ...string) string {
	return filepath.Join(paths...)
}

// IsWindows 检查当前操作系统是否为Windows
func IsWindows() bool {
	return runtime.GOOS == "windows"
}
