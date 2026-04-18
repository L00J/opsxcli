package ssh

import (
	"fmt"
	"strings"
)

// parseTarget 解析目标地址 user@host
func parseTarget(target string) (user, host string, err error) {
	parts := strings.Split(target, "@")
	if len(parts) == 1 {
		return "root", parts[0], nil
	}
	if len(parts) == 2 {
		return parts[0], parts[1], nil
	}
	return "", "", fmt.Errorf("无效的目标格式: %s", target)
}

// parseRemotePath 解析远程路径 user@host:/path
func parseRemotePath(remote string) (user, host, path string, err error) {
	if !strings.Contains(remote, ":") {
		return "", "", "", fmt.Errorf("无效的远程路径格式: %s (需要 user@host:/path)", remote)
	}

	idx := strings.LastIndex(remote, ":")
	userHost := remote[:idx]
	path = remote[idx+1:]

	user, host, err = parseTarget(userHost)
	if err != nil {
		return "", "", "", err
	}

	return user, host, path, nil
}
