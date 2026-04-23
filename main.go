package main

import (
	"os"
	"strings"

	"opsxcli/cmd"
	"opsxcli/internal/logger"

	"github.com/spf13/cobra"
)

var (
	version   = "dev"
	buildTime = "unknown"
	gitCommit = "unknown"
)

// knownCommands 已注册的命令列表（用于默认 Agent 查询判断）
var knownCommands = map[string]bool{
	"agent": true, "session": true,
	"mysql": true, "mysqldump": true, "mysqlrestore": true,
	"psql": true, "pgdump": true, "pgrestore": true,
	"redis": true,
	"ssh":   true, "ssh-config": true, "telnet": true, "nc": true, "ping": true,
	"traceroute": true, "netstat": true, "ss": true, "nmap": true,
	"sys": true, "net": true,
	"server": true,
	"curl":   true, "wget": true, "request": true, "websearch": true,
	"kubectl": true, "consul": true, "kubernetes": true, "test-db": true,
	"docker":  true,
	"install": true, "upgrade": true,
	"ls": true, "cp": true, "mv": true, "rm": true, "mkdir": true,
	"rmdir": true, "touch": true, "chmod": true, "chown": true, "ln": true,
	"cat": true, "head": true, "tail": true, "grep": true,
	"tree": true,
	"ps":   true, "top": true, "kill": true, "pstree": true, "dd": true,
	"ifconfig": true, "route": true, "ip": true,
	"tar": true, "gzip": true, "unzip": true,
	"df": true, "du": true, "free": true,
	"uname": true, "hostname": true, "whoami": true, "id": true,
	"date": true, "sleep": true,
	"help": true, "version": true,
	"completion": true, "search": true,
	"logs": true,
	"ssl":  true, "bench": true, "notify": true,
	"setup": true,
}

// removeHelpFlagShorthand 移除子命令的 help flag 的 shorthand，避免与自定义 flags 冲突
// 注意：根命令的 -h 保留，只移除子命令的 -h
func removeHelpFlagShorthand(c *cobra.Command) {
	// 递归处理所有子命令
	for _, subCmd := range c.Commands() {
		// 递归处理子命令的子命令
		removeHelpFlagShorthand(subCmd)

		// 移除子命令的 help flag shorthand
		if helpFlag := subCmd.PersistentFlags().Lookup("help"); helpFlag != nil {
			helpFlag.Shorthand = ""
			helpFlag.Hidden = true
		}
		if helpFlag := subCmd.Flags().Lookup("help"); helpFlag != nil {
			helpFlag.Shorthand = ""
			helpFlag.Hidden = true
		}
	}
}

// preprocessDatabasePasswordArgs 预处理数据库命令的密码参数
// 将 -pPASSWORD 格式转换为 -p PASSWORD 以兼容 Cobra flag 解析
func preprocessDatabasePasswordArgs() {
	if len(os.Args) < 2 {
		return
	}

	// 检查是否是数据库命令 (mysql, psql, redis)
	dbCommands := map[string]bool{"mysql": true, "psql": true, "redis": true}
	if !dbCommands[os.Args[1]] {
		return
	}

	// 查找并处理 -pXXX 格式的密码参数
	for i := 2; i < len(os.Args); i++ {
		arg := os.Args[i]

		// 匹配 -pXXX 格式（密码紧跟在 -p 后面）
		if strings.HasPrefix(arg, "-p") && len(arg) > 2 && arg[2] != '-' {
			pwd := arg[2:]
			cmd.SetDatabasePassword(pwd)

			// 将 -pPASSWORD 替换为 -p 和 PASSWORD 两个参数
			os.Args[i] = "-p"
			os.Args = append(os.Args[:i+1], append([]string{pwd}, os.Args[i+1:]...)...)
			break
		}
	}
}

// preprocessAgentQueryArgs 预处理默认 Agent 查询
// 当第一个参数不是已知命令时，自动转换为 agent -q "query"
func preprocessAgentQueryArgs() {
	if len(os.Args) < 2 {
		return
	}

	firstArg := os.Args[1]

	// 如果以 - 开头，是 flag，不处理
	if strings.HasPrefix(firstArg, "-") {
		return
	}

	// 检查是否是已知命令
	if knownCommands[firstArg] {
		return
	}

	// 不是已知命令，自动转换为 agent -q "query"
	query := strings.Join(os.Args[1:], " ")
	os.Args = []string{os.Args[0], "agent", "-q", query}
}

func main() {
	// 预处理默认 Agent 查询（必须在数据库密码处理之前）
	// 示例: opsxcli "磁盘使用率" → opsxcli agent -q "磁盘使用率"
	preprocessAgentQueryArgs()

	// 预处理数据库密码参数（必须在创建命令之前）
	// 将 -pPASSWORD 拆分为 -p PASSWORD 避免 Cobra 解析错误
	preprocessDatabasePasswordArgs()

	// 初始化日志系统
	logger.Init()
	defer logger.Close()

	// 创建根命令
	rootCmd := cmd.NewRootCmd(version)

	// 移除所有命令（包括子命令）的 help flag 短选项
	// 必须在 Execute 之前处理，确保 -h 可以用于 host 等参数
	removeHelpFlagShorthand(rootCmd)

	// 执行命令
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
