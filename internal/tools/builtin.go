package tools

import (
	"fmt"
	"strings"
)

// RegisterDefaultTools 注册默认工具
func RegisterDefaultTools(registry *ToolRegistry) {
	// Web搜索工具
	registry.Register(NewWebSearchTool())

	// 文件查看工具
	registry.Register(NewCatTool())
	registry.Register(NewGrepTool())
	registry.Register(NewLsTool())
	registry.Register(NewHeadTool())
	registry.Register(NewTailTool())
	registry.Register(NewTreeTool())

	// 文件/目录管理工具
	registry.Register(NewMkdirTool())
	registry.Register(NewTouchTool())
	registry.Register(NewChmodTool())
	registry.Register(NewChownTool())

	// 系统信息工具
	registry.Register(NewPsTool())
	registry.Register(NewTopTool())
	registry.Register(NewFreeTool())
	registry.Register(NewDfTool())
	registry.Register(NewDuTool())
	registry.Register(NewUnameTool())

	// 网络工具
	registry.Register(NewPingTool())
	registry.Register(NewSsTool())
	registry.Register(NewNetstatTool())
	registry.Register(NewIfconfigTool())
	registry.Register(NewWgetTool())
	registry.Register(NewCurlTool())

	// SSH工具
	registry.Register(NewSSHTool())

	// 系统管理工具
	registry.Register(NewInstallTool())
	registry.Register(NewSmartBashTool()) // 使用智能bash工具

	// Kubectl工具
	registry.Register(NewKubectlGetTool())
	registry.Register(NewKubectlDescribeTool())
	registry.Register(NewKubectlLogsTool())

	// 代码开发工具
	registry.Register(NewFileReadTool())
	registry.Register(NewFileWriteTool())
	registry.Register(NewFileEditTool())
	registry.Register(NewGitStatusTool())
	registry.Register(NewGitDiffTool())
	registry.Register(NewCodeSearchTool())

	// 危险操作（需要审批）
	registry.Register(NewRmTool())
	registry.Register(NewDdTool())
	registry.Register(NewKubectlDeleteTool())
}

// NewCatTool 创建cat工具
func NewCatTool() *CommandTool {
	return NewCommandTool(
		"cat",
		"查看文件内容",
		map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"file": map[string]interface{}{
					"type":        "string",
					"description": "文件路径",
				},
			},
			"required": []string{"file"},
		},
		RiskSafe,
		func(args map[string]interface{}) (string, []string, error) {
			file := parseStringParam(args, "file")
			if file == "" {
				return "", nil, fmt.Errorf("file 参数不能为空")
			}
			return "opsxcli", []string{"cat", file}, nil
		},
	)
}

// NewGrepTool 创建grep工具
func NewGrepTool() *CommandTool {
	return NewCommandTool(
		"grep",
		"在文件中搜索文本模式",
		map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"pattern": map[string]interface{}{
					"type":        "string",
					"description": "搜索模式",
				},
				"file": map[string]interface{}{
					"type":        "string",
					"description": "文件路径",
				},
				"recursive": map[string]interface{}{
					"type":        "boolean",
					"description": "递归搜索目录",
				},
			},
			"required": []string{"pattern", "file"},
		},
		RiskSafe,
		func(args map[string]interface{}) (string, []string, error) {
			pattern := parseStringParam(args, "pattern")
			file := parseStringParam(args, "file")
			recursive := parseBoolParam(args, "recursive")

			if pattern == "" || file == "" {
				return "", nil, fmt.Errorf("pattern 和 file 参数不能为空")
			}

			cmdArgs := []string{"grep", pattern, file}
			if recursive {
				cmdArgs = []string{"grep", "-r", pattern, file}
			}

			return "opsxcli", cmdArgs, nil
		},
	)
}

// NewLsTool 创建ls工具
func NewLsTool() *CommandTool {
	return NewCommandTool(
		"ls",
		"列出目录内容",
		map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"path": map[string]interface{}{
					"type":        "string",
					"description": "目录路径（默认为当前目录）",
				},
				"all": map[string]interface{}{
					"type":        "boolean",
					"description": "显示隐藏文件",
				},
				"long": map[string]interface{}{
					"type":        "boolean",
					"description": "详细信息格式",
				},
			},
		},
		RiskSafe,
		func(args map[string]interface{}) (string, []string, error) {
			path := parseStringParam(args, "path")
			if path == "" {
				path = "."
			}

			cmdArgs := []string{"ls"}
			if parseBoolParam(args, "all") {
				cmdArgs = append(cmdArgs, "-a")
			}
			if parseBoolParam(args, "long") {
				cmdArgs = append(cmdArgs, "-l")
			}
			cmdArgs = append(cmdArgs, path)

			return "opsxcli", cmdArgs, nil
		},
	)
}

// NewHeadTool 创建head工具
func NewHeadTool() *CommandTool {
	return NewCommandTool(
		"head",
		"显示文件开头部分",
		map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"file": map[string]interface{}{
					"type":        "string",
					"description": "文件路径",
				},
				"lines": map[string]interface{}{
					"type":        "number",
					"description": "显示行数（默认10行）",
				},
			},
			"required": []string{"file"},
		},
		RiskSafe,
		func(args map[string]interface{}) (string, []string, error) {
			file := parseStringParam(args, "file")
			lines := parseIntParam(args, "lines")

			if file == "" {
				return "", nil, fmt.Errorf("file 参数不能为空")
			}

			cmdArgs := []string{"head"}
			if lines > 0 {
				cmdArgs = append(cmdArgs, "-n", fmt.Sprintf("%d", lines))
			}
			cmdArgs = append(cmdArgs, file)

			return "opsxcli", cmdArgs, nil
		},
	)
}

// NewTailTool 创建tail工具
func NewTailTool() *CommandTool {
	return NewCommandTool(
		"tail",
		"显示文件末尾部分",
		map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"file": map[string]interface{}{
					"type":        "string",
					"description": "文件路径",
				},
				"lines": map[string]interface{}{
					"type":        "number",
					"description": "显示行数（默认10行）",
				},
				"follow": map[string]interface{}{
					"type":        "boolean",
					"description": "持续监控文件变化",
				},
			},
			"required": []string{"file"},
		},
		RiskSafe,
		func(args map[string]interface{}) (string, []string, error) {
			file := parseStringParam(args, "file")
			lines := parseIntParam(args, "lines")
			follow := parseBoolParam(args, "follow")

			if file == "" {
				return "", nil, fmt.Errorf("file 参数不能为空")
			}

			cmdArgs := []string{"tail"}
			if lines > 0 {
				cmdArgs = append(cmdArgs, "-n", fmt.Sprintf("%d", lines))
			}
			if follow {
				cmdArgs = append(cmdArgs, "-f")
			}
			cmdArgs = append(cmdArgs, file)

			return "opsxcli", cmdArgs, nil
		},
	)
}

// NewTreeTool 创建tree工具
func NewTreeTool() *CommandTool {
	return NewCommandTool(
		"tree",
		"以树形结构显示目录",
		map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"path": map[string]interface{}{
					"type":        "string",
					"description": "目录路径",
				},
				"level": map[string]interface{}{
					"type":        "number",
					"description": "显示层级深度",
				},
			},
		},
		RiskSafe,
		func(args map[string]interface{}) (string, []string, error) {
			path := parseStringParam(args, "path")
			if path == "" {
				path = "."
			}

			cmdArgs := []string{"tree"}
			if level := parseIntParam(args, "level"); level > 0 {
				cmdArgs = append(cmdArgs, "-L", fmt.Sprintf("%d", level))
			}
			cmdArgs = append(cmdArgs, path)

			return "opsxcli", cmdArgs, nil
		},
	)
}

// NewPsTool 创建ps工具
func NewPsTool() *CommandTool {
	return NewCommandTool(
		"ps",
		"查看进程列表",
		map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"all": map[string]interface{}{
					"type":        "boolean",
					"description": "显示所有进程",
				},
			},
		},
		RiskSafe,
		func(args map[string]interface{}) (string, []string, error) {
			cmdArgs := []string{"ps"}
			if parseBoolParam(args, "all") {
				cmdArgs = append(cmdArgs, "aux")
			}
			return "opsxcli", cmdArgs, nil
		},
	)
}

// NewTopTool 创建top工具
func NewTopTool() *CommandTool {
	return NewCommandTool(
		"top",
		"实时显示进程动态",
		map[string]interface{}{
			"type":       "object",
			"properties": map[string]interface{}{},
		},
		RiskSafe,
		func(args map[string]interface{}) (string, []string, error) {
			// top命令需要特殊处理，这里简化为ps
			return "opsxcli", []string{"ps", "aux"}, nil
		},
	)
}

// NewFreeTool 创建free工具
func NewFreeTool() *CommandTool {
	return NewCommandTool(
		"free",
		"显示内存使用情况",
		map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"human": map[string]interface{}{
					"type":        "boolean",
					"description": "人类可读格式",
				},
			},
		},
		RiskSafe,
		func(args map[string]interface{}) (string, []string, error) {
			cmdArgs := []string{"free"}
			if parseBoolParam(args, "human") {
				cmdArgs = append(cmdArgs, "-h")
			}
			return "opsxcli", cmdArgs, nil
		},
	)
}

// NewDfTool 创建df工具
func NewDfTool() *CommandTool {
	return NewCommandTool(
		"df",
		"显示磁盘空间使用情况",
		map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"human": map[string]interface{}{
					"type":        "boolean",
					"description": "人类可读格式",
				},
			},
		},
		RiskSafe,
		func(args map[string]interface{}) (string, []string, error) {
			cmdArgs := []string{"df"}
			if parseBoolParam(args, "human") {
				cmdArgs = append(cmdArgs, "-h")
			}
			return "opsxcli", cmdArgs, nil
		},
	)
}

// NewDuTool 创建du工具
func NewDuTool() *CommandTool {
	return NewCommandTool(
		"du",
		"显示文件/目录大小",
		map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"path": map[string]interface{}{
					"type":        "string",
					"description": "路径",
				},
				"human": map[string]interface{}{
					"type":        "boolean",
					"description": "人类可读格式",
				},
			},
			"required": []string{"path"},
		},
		RiskSafe,
		func(args map[string]interface{}) (string, []string, error) {
			path := parseStringParam(args, "path")
			if path == "" {
				return "", nil, fmt.Errorf("path 参数不能为空")
			}

			cmdArgs := []string{"du"}
			if parseBoolParam(args, "human") {
				cmdArgs = append(cmdArgs, "-h")
			}
			cmdArgs = append(cmdArgs, path)

			return "opsxcli", cmdArgs, nil
		},
	)
}

// NewUnameTool 创建uname工具
func NewUnameTool() *CommandTool {
	return NewCommandTool(
		"uname",
		"显示系统信息",
		map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"all": map[string]interface{}{
					"type":        "boolean",
					"description": "显示所有信息",
				},
			},
		},
		RiskSafe,
		func(args map[string]interface{}) (string, []string, error) {
			cmdArgs := []string{"uname"}
			if parseBoolParam(args, "all") {
				cmdArgs = append(cmdArgs, "-a")
			}
			return "opsxcli", cmdArgs, nil
		},
	)
}

// NewPingTool 创建ping工具
func NewPingTool() *CommandTool {
	return NewCommandTool(
		"ping",
		"测试网络连通性",
		map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"host": map[string]interface{}{
					"type":        "string",
					"description": "目标主机",
				},
				"count": map[string]interface{}{
					"type":        "number",
					"description": "发送包数量（默认4）",
				},
			},
			"required": []string{"host"},
		},
		RiskSafe,
		func(args map[string]interface{}) (string, []string, error) {
			host := parseStringParam(args, "host")
			if host == "" {
				return "", nil, fmt.Errorf("host 参数不能为空")
			}

			count := parseIntParam(args, "count")
			if count == 0 {
				count = 4
			}

			return "opsxcli", []string{"ping", "-c", fmt.Sprintf("%d", count), host}, nil
		},
	)
}

// NewSsTool 创建ss工具
func NewSsTool() *CommandTool {
	return NewCommandTool(
		"ss",
		"查看网络连接状态",
		map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"listening": map[string]interface{}{
					"type":        "boolean",
					"description": "只显示监听端口",
				},
				"tcp": map[string]interface{}{
					"type":        "boolean",
					"description": "只显示TCP连接",
				},
			},
		},
		RiskSafe,
		func(args map[string]interface{}) (string, []string, error) {
			cmdArgs := []string{"ss", "-n"}
			if parseBoolParam(args, "listening") {
				cmdArgs = append(cmdArgs, "-l")
			}
			if parseBoolParam(args, "tcp") {
				cmdArgs = append(cmdArgs, "-t")
			}
			return "opsxcli", cmdArgs, nil
		},
	)
}

// NewNetstatTool 创建netstat工具
func NewNetstatTool() *CommandTool {
	return NewCommandTool(
		"netstat",
		"查看网络状态",
		map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"listening": map[string]interface{}{
					"type":        "boolean",
					"description": "只显示监听端口",
				},
			},
		},
		RiskSafe,
		func(args map[string]interface{}) (string, []string, error) {
			cmdArgs := []string{"netstat", "-n"}
			if parseBoolParam(args, "listening") {
				cmdArgs = append(cmdArgs, "-l")
			}
			return "opsxcli", cmdArgs, nil
		},
	)
}

// NewIfconfigTool 创建ifconfig工具
func NewIfconfigTool() *CommandTool {
	return NewCommandTool(
		"ifconfig",
		"显示网络接口信息",
		map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"interface": map[string]interface{}{
					"type":        "string",
					"description": "网络接口名称",
				},
			},
		},
		RiskSafe,
		func(args map[string]interface{}) (string, []string, error) {
			cmdArgs := []string{"ifconfig"}
			if iface := parseStringParam(args, "interface"); iface != "" {
				cmdArgs = append(cmdArgs, iface)
			}
			return "opsxcli", cmdArgs, nil
		},
	)
}

// NewSSHTool 创建ssh工具
func NewSSHTool() *CommandTool {
	return NewCommandTool(
		"ssh",
		"SSH远程连接并执行命令",
		map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"host": map[string]interface{}{
					"type":        "string",
					"description": "目标主机（user@host:port格式）",
				},
				"command": map[string]interface{}{
					"type":        "string",
					"description": "要执行的命令",
				},
			},
			"required": []string{"host", "command"},
		},
		RiskMedium,
		func(args map[string]interface{}) (string, []string, error) {
			host := parseStringParam(args, "host")
			command := parseStringParam(args, "command")

			if host == "" || command == "" {
				return "", nil, fmt.Errorf("host 和 command 参数不能为空")
			}

			return "opsxcli", []string{"ssh", host, "-c", command}, nil
		},
	)
}

// NewKubectlGetTool 创建kubectl get工具
func NewKubectlGetTool() *CommandTool {
	return NewCommandTool(
		"kubectl_get",
		"获取Kubernetes资源",
		map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"resource": map[string]interface{}{
					"type":        "string",
					"description": "资源类型（pods, services, deployments等）",
				},
				"namespace": map[string]interface{}{
					"type":        "string",
					"description": "命名空间",
				},
				"name": map[string]interface{}{
					"type":        "string",
					"description": "资源名称",
				},
			},
			"required": []string{"resource"},
		},
		RiskSafe,
		func(args map[string]interface{}) (string, []string, error) {
			resource := parseStringParam(args, "resource")
			namespace := parseStringParam(args, "namespace")
			name := parseStringParam(args, "name")

			if resource == "" {
				return "", nil, fmt.Errorf("resource 参数不能为空")
			}

			cmdArgs := []string{"kubectl", "get", resource}
			if namespace != "" {
				cmdArgs = append(cmdArgs, "-n", namespace)
			}
			if name != "" {
				cmdArgs = append(cmdArgs, name)
			}
			cmdArgs = append(cmdArgs, "-o", "wide")

			return "opsxcli", cmdArgs, nil
		},
	)
}

// NewKubectlDescribeTool 创建kubectl describe工具
func NewKubectlDescribeTool() *CommandTool {
	return NewCommandTool(
		"kubectl_describe",
		"查看Kubernetes资源详细信息",
		map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"resource": map[string]interface{}{
					"type":        "string",
					"description": "资源类型",
				},
				"name": map[string]interface{}{
					"type":        "string",
					"description": "资源名称",
				},
				"namespace": map[string]interface{}{
					"type":        "string",
					"description": "命名空间",
				},
			},
			"required": []string{"resource", "name"},
		},
		RiskSafe,
		func(args map[string]interface{}) (string, []string, error) {
			resource := parseStringParam(args, "resource")
			name := parseStringParam(args, "name")
			namespace := parseStringParam(args, "namespace")

			if resource == "" || name == "" {
				return "", nil, fmt.Errorf("resource 和 name 参数不能为空")
			}

			cmdArgs := []string{"kubectl", "describe", resource, name}
			if namespace != "" {
				cmdArgs = append(cmdArgs, "-n", namespace)
			}

			return "opsxcli", cmdArgs, nil
		},
	)
}

// NewKubectlLogsTool 创建kubectl logs工具
func NewKubectlLogsTool() *CommandTool {
	return NewCommandTool(
		"kubectl_logs",
		"查看Pod日志",
		map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"pod": map[string]interface{}{
					"type":        "string",
					"description": "Pod名称",
				},
				"namespace": map[string]interface{}{
					"type":        "string",
					"description": "命名空间",
				},
				"container": map[string]interface{}{
					"type":        "string",
					"description": "容器名称",
				},
				"tail": map[string]interface{}{
					"type":        "number",
					"description": "显示最后N行",
				},
			},
			"required": []string{"pod"},
		},
		RiskSafe,
		func(args map[string]interface{}) (string, []string, error) {
			pod := parseStringParam(args, "pod")
			namespace := parseStringParam(args, "namespace")
			container := parseStringParam(args, "container")
			tail := parseIntParam(args, "tail")

			if pod == "" {
				return "", nil, fmt.Errorf("pod 参数不能为空")
			}

			cmdArgs := []string{"kubectl", "logs", pod}
			if namespace != "" {
				cmdArgs = append(cmdArgs, "-n", namespace)
			}
			if container != "" {
				cmdArgs = append(cmdArgs, "-c", container)
			}
			if tail > 0 {
				cmdArgs = append(cmdArgs, "--tail", fmt.Sprintf("%d", tail))
			}

			return "opsxcli", cmdArgs, nil
		},
	)
}

// NewRmTool 创建rm工具（危险操作）
func NewRmTool() *CommandTool {
	return NewCommandTool(
		"rm",
		"删除文件或目录",
		map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"path": map[string]interface{}{
					"type":        "string",
					"description": "要删除的路径",
				},
				"recursive": map[string]interface{}{
					"type":        "boolean",
					"description": "递归删除目录",
				},
				"force": map[string]interface{}{
					"type":        "boolean",
					"description": "强制删除",
				},
			},
			"required": []string{"path"},
		},
		RiskCritical,
		func(args map[string]interface{}) (string, []string, error) {
			path := parseStringParam(args, "path")
			if path == "" {
				return "", nil, fmt.Errorf("path 参数不能为空")
			}

			// 防止删除根目录
			if path == "/" || path == "/*" {
				return "", nil, fmt.Errorf("禁止删除根目录")
			}

			cmdArgs := []string{"rm"}
			if parseBoolParam(args, "recursive") {
				cmdArgs = append(cmdArgs, "-r")
			}
			if parseBoolParam(args, "force") {
				cmdArgs = append(cmdArgs, "-f")
			}
			cmdArgs = append(cmdArgs, path)

			return "opsxcli", cmdArgs, nil
		},
	)
}

// NewDdTool 创建dd工具（危险操作）
func NewDdTool() *CommandTool {
	return NewCommandTool(
		"dd",
		"转换和复制文件",
		map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"if": map[string]interface{}{
					"type":        "string",
					"description": "输入文件",
				},
				"of": map[string]interface{}{
					"type":        "string",
					"description": "输出文件",
				},
				"bs": map[string]interface{}{
					"type":        "string",
					"description": "块大小",
				},
				"count": map[string]interface{}{
					"type":        "number",
					"description": "块数量",
				},
			},
			"required": []string{"if", "of"},
		},
		RiskCritical,
		func(args map[string]interface{}) (string, []string, error) {
			inputFile := parseStringParam(args, "if")
			outputFile := parseStringParam(args, "of")

			if inputFile == "" || outputFile == "" {
				return "", nil, fmt.Errorf("if 和 of 参数不能为空")
			}

			// 检查是否是块设备
			if strings.HasPrefix(outputFile, "/dev/") {
				return "", nil, fmt.Errorf("禁止直接写入块设备，风险过高")
			}

			cmdArgs := []string{"dd", "if=" + inputFile, "of=" + outputFile}
			if bs := parseStringParam(args, "bs"); bs != "" {
				cmdArgs = append(cmdArgs, "bs="+bs)
			}
			if count := parseIntParam(args, "count"); count > 0 {
				cmdArgs = append(cmdArgs, fmt.Sprintf("count=%d", count))
			}

			return "opsxcli", cmdArgs, nil
		},
	)
}

// NewKubectlDeleteTool 创建kubectl delete工具（危险操作）
func NewKubectlDeleteTool() *CommandTool {
	return NewCommandTool(
		"kubectl_delete",
		"删除Kubernetes资源",
		map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"resource": map[string]interface{}{
					"type":        "string",
					"description": "资源类型",
				},
				"name": map[string]interface{}{
					"type":        "string",
					"description": "资源名称",
				},
				"namespace": map[string]interface{}{
					"type":        "string",
					"description": "命名空间",
				},
			},
			"required": []string{"resource", "name"},
		},
		RiskHigh,
		func(args map[string]interface{}) (string, []string, error) {
			resource := parseStringParam(args, "resource")
			name := parseStringParam(args, "name")
			namespace := parseStringParam(args, "namespace")

			if resource == "" || name == "" {
				return "", nil, fmt.Errorf("resource 和 name 参数不能为空")
			}

			cmdArgs := []string{"kubectl", "delete", resource, name}
			if namespace != "" {
				cmdArgs = append(cmdArgs, "-n", namespace)
			}

			return "opsxcli", cmdArgs, nil
		},
	)
}

// NewInstallTool 创建install工具
func NewInstallTool() *CommandTool {
	return NewCommandTool(
		"install",
		"使用系统包管理器安装软件包(自动检测yum/dnf/apt/brew)",
		map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"package": map[string]interface{}{
					"type":        "string",
					"description": "软件包名称(如mysql, nginx, redis, java, rocketmq等)",
				},
			},
			"required": []string{"package"},
		},
		RiskHigh,
		func(args map[string]interface{}) (string, []string, error) {
			packageName := parseStringParam(args, "package")

			if packageName == "" {
				return "", nil, fmt.Errorf("package 参数不能为空")
			}

			return "opsxcli", []string{"install", packageName}, nil
		},
	)
}

// NewMkdirTool 创建mkdir工具
func NewMkdirTool() *CommandTool {
	return NewCommandTool(
		"mkdir",
		"创建目录",
		map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"path": map[string]interface{}{
					"type":        "string",
					"description": "目录路径",
				},
				"parents": map[string]interface{}{
					"type":        "boolean",
					"description": "递归创建父目录(-p)",
				},
			},
			"required": []string{"path"},
		},
		RiskLow,
		func(args map[string]interface{}) (string, []string, error) {
			path := parseStringParam(args, "path")
			if path == "" {
				return "", nil, fmt.Errorf("path 参数不能为空")
			}

			cmdArgs := []string{"mkdir"}
			if parseBoolParam(args, "parents") {
				cmdArgs = append(cmdArgs, "-p")
			}
			cmdArgs = append(cmdArgs, path)

			return "opsxcli", cmdArgs, nil
		},
	)
}

// NewChmodTool 创建chmod工具
func NewChmodTool() *CommandTool {
	return NewCommandTool(
		"chmod",
		"修改文件/目录权限",
		map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"mode": map[string]interface{}{
					"type":        "string",
					"description": "权限模式(如755, 644, +x等)",
				},
				"path": map[string]interface{}{
					"type":        "string",
					"description": "文件或目录路径",
				},
				"recursive": map[string]interface{}{
					"type":        "boolean",
					"description": "递归修改(-R)",
				},
			},
			"required": []string{"mode", "path"},
		},
		RiskMedium,
		func(args map[string]interface{}) (string, []string, error) {
			mode := parseStringParam(args, "mode")
			path := parseStringParam(args, "path")

			if mode == "" || path == "" {
				return "", nil, fmt.Errorf("mode 和 path 参数不能为空")
			}

			cmdArgs := []string{"chmod"}
			if parseBoolParam(args, "recursive") {
				cmdArgs = append(cmdArgs, "-R")
			}
			cmdArgs = append(cmdArgs, mode, path)

			return "opsxcli", cmdArgs, nil
		},
	)
}

// NewChownTool 创建chown工具
func NewChownTool() *CommandTool {
	return NewCommandTool(
		"chown",
		"修改文件/目录所有者",
		map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"owner": map[string]interface{}{
					"type":        "string",
					"description": "所有者(user:group格式)",
				},
				"path": map[string]interface{}{
					"type":        "string",
					"description": "文件或目录路径",
				},
				"recursive": map[string]interface{}{
					"type":        "boolean",
					"description": "递归修改(-R)",
				},
			},
			"required": []string{"owner", "path"},
		},
		RiskHigh,
		func(args map[string]interface{}) (string, []string, error) {
			owner := parseStringParam(args, "owner")
			path := parseStringParam(args, "path")

			if owner == "" || path == "" {
				return "", nil, fmt.Errorf("owner 和 path 参数不能为空")
			}

			cmdArgs := []string{"chown"}
			if parseBoolParam(args, "recursive") {
				cmdArgs = append(cmdArgs, "-R")
			}
			cmdArgs = append(cmdArgs, owner, path)

			return "opsxcli", cmdArgs, nil
		},
	)
}

// NewTouchTool 创建touch工具
func NewTouchTool() *CommandTool {
	return NewCommandTool(
		"touch",
		"创建空文件或更新文件时间戳",
		map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"path": map[string]interface{}{
					"type":        "string",
					"description": "文件路径",
				},
			},
			"required": []string{"path"},
		},
		RiskLow,
		func(args map[string]interface{}) (string, []string, error) {
			path := parseStringParam(args, "path")
			if path == "" {
				return "", nil, fmt.Errorf("path 参数不能为空")
			}

			return "opsxcli", []string{"touch", path}, nil
		},
	)
}

// NewWgetTool 创建wget工具
func NewWgetTool() *CommandTool {
	return NewCommandTool(
		"wget",
		"下载文件",
		map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"url": map[string]interface{}{
					"type":        "string",
					"description": "下载链接",
				},
				"output": map[string]interface{}{
					"type":        "string",
					"description": "输出文件名(-O)",
				},
				"directory": map[string]interface{}{
					"type":        "string",
					"description": "保存目录(-P)",
				},
			},
			"required": []string{"url"},
		},
		RiskMedium,
		func(args map[string]interface{}) (string, []string, error) {
			url := parseStringParam(args, "url")
			if url == "" {
				return "", nil, fmt.Errorf("url 参数不能为空")
			}

			cmdArgs := []string{"wget"}
			if output := parseStringParam(args, "output"); output != "" {
				cmdArgs = append(cmdArgs, "-O", output)
			}
			if dir := parseStringParam(args, "directory"); dir != "" {
				cmdArgs = append(cmdArgs, "-P", dir)
			}
			cmdArgs = append(cmdArgs, url)

			return "opsxcli", cmdArgs, nil
		},
	)
}

// NewCurlTool 创建curl工具
func NewCurlTool() *CommandTool {
	return NewCommandTool(
		"curl",
		"HTTP请求工具",
		map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"url": map[string]interface{}{
					"type":        "string",
					"description": "请求URL",
				},
				"method": map[string]interface{}{
					"type":        "string",
					"description": "HTTP方法(GET, POST, PUT, DELETE等)",
				},
				"data": map[string]interface{}{
					"type":        "string",
					"description": "POST数据",
				},
				"header": map[string]interface{}{
					"type":        "string",
					"description": "HTTP头(可以多个,用;分隔)",
				},
				"output": map[string]interface{}{
					"type":        "string",
					"description": "输出到文件(-o)",
				},
			},
			"required": []string{"url"},
		},
		RiskSafe,
		func(args map[string]interface{}) (string, []string, error) {
			url := parseStringParam(args, "url")
			if url == "" {
				return "", nil, fmt.Errorf("url 参数不能为空")
			}

			cmdArgs := []string{"curl"}
			
			if method := parseStringParam(args, "method"); method != "" {
				cmdArgs = append(cmdArgs, "-X", method)
			}
			
			if data := parseStringParam(args, "data"); data != "" {
				cmdArgs = append(cmdArgs, "-d", data)
			}
			
			if header := parseStringParam(args, "header"); header != "" {
				headers := strings.Split(header, ";")
				for _, h := range headers {
					if h != "" {
						cmdArgs = append(cmdArgs, "-H", strings.TrimSpace(h))
					}
				}
			}
			
			if output := parseStringParam(args, "output"); output != "" {
				cmdArgs = append(cmdArgs, "-o", output)
			}
			
			cmdArgs = append(cmdArgs, url)

			return "opsxcli", cmdArgs, nil
		},
	)
}

// NewBashTool 创建bash工具(可执行任意shell命令)
func NewBashTool() *CommandTool {
	return NewCommandTool(
		"bash",
		"执行Shell命令或脚本(强大但有风险)",
		map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"command": map[string]interface{}{
					"type":        "string",
					"description": "要执行的shell命令或脚本内容",
				},
				"script": map[string]interface{}{
					"type":        "boolean",
					"description": "是否作为脚本执行(通过临时文件)",
				},
			},
			"required": []string{"command"},
		},
		RiskHigh,
		func(args map[string]interface{}) (string, []string, error) {
			command := parseStringParam(args, "command")
			if command == "" {
				return "", nil, fmt.Errorf("command 参数不能为空")
			}

			// 如果是脚本模式,创建临时文件
			if parseBoolParam(args, "script") {
				return "bash", []string{"-c", command}, nil
			}

			// 直接执行命令
			return "bash", []string{"-c", command}, nil
		},
	)
}
