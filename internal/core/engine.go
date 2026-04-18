package core

import (
	"fmt"
	"runtime"
)

// BuildInfo 包含编译时注入的构建信息
type BuildInfo struct {
	Version   string
	BuildTime string
	GitCommit string
}

// NewBuildInfo 创建构建信息
func NewBuildInfo(version, buildTime, gitCommit string) *BuildInfo {
	return &BuildInfo{
		Version:   version,
		BuildTime: buildTime,
		GitCommit: gitCommit,
	}
}

// String 返回格式化的版本字符串
func (b *BuildInfo) String() string {
	if b.GitCommit != "" && b.GitCommit != "unknown" {
		return fmt.Sprintf("%s (commit: %s)", b.Version, b.GitCommit)
	}
	return b.Version
}

// All 返回完整的构建信息映射
func (b *BuildInfo) All() map[string]string {
	return map[string]string{
		"version":   b.Version,
		"buildTime": b.BuildTime,
		"gitCommit": b.GitCommit,
		"platform":  fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH),
		"goVersion": runtime.Version(),
	}
}
