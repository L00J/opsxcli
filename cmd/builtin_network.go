package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"opsxcli/plugins/builtin"
)

func init() {
	RegisterCommand("ifconfig", "系统", "网络接口配置", NewIfconfigCmd)
	RegisterCommand("route", "系统", "路由表管理", NewRouteCmd)
	RegisterCommand("ip", "系统", "IP地址管理", NewIpCmd)
}

// === 网络配置 ===

func NewIfconfigCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "ifconfig",
		Short: "网络接口配置",
		Long:  "显示网络接口配置信息",
		RunE: func(cmd *cobra.Command, args []string) error {
			return builtin.ShowInterfaces()
		},
	}
}

func NewRouteCmd() *cobra.Command {
	return &cobra.Command{
		Use:                "route",
		Short:              "显示或修改路由表",
		Long:               "显示内核 IP 路由表",
		DisableFlagParsing: true, // 禁用参数解析
		RunE: func(cmd *cobra.Command, args []string) error {
			return builtin.ShowRoutes()
		},
	}
}

func NewIpCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "ip [ OPTIONS ] OBJECT { COMMAND | help }",
		Short: "显示/操作路由、网络设备、接口和隧道",
		Long: `ip - 显示/操作路由、网络设备、接口和隧道

用法: ip [ OPTIONS ] OBJECT { COMMAND | help }
      ip [ -force ] -batch filename

OBJECT := { link | address | route | help }
OPTIONS := { -V[ersion] | -h[uman-readable] | -s[tatistics] |
             -r[esolve] | -f[amily] { inet | inet6 } |
             -4 | -6 | -o[neline] | -br[ief] }

常用命令:
  ip addr           显示所有网络接口的 IP 地址
  ip addr show      显示所有网络接口的 IP 地址
  ip link           显示所有网络接口信息
  ip link show      显示所有网络接口信息
  ip route          显示路由表
  ip route show     显示路由表

简写形式:
  ip a              等同于 ip addr
  ip l              等同于 ip link
  ip r              等同于 ip route`,
		DisableFlagParsing: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return handleIPCommand(args)
		},
	}
}

// handleIPCommand 处理 ip 命令的各种子命令
func handleIPCommand(args []string) error {
	// 没有参数，默认显示 addr
	if len(args) == 0 {
		return builtin.ShowIPAddr()
	}

	// 过滤掉选项参数（以 - 开头的）
	var object string
	for i, arg := range args {
		if arg == "-h" || arg == "--help" || arg == "help" {
			// 显示帮助信息
			fmt.Println(getIPHelpText())
			return nil
		}
		if !strings.HasPrefix(arg, "-") {
			object = arg
			break
		}
		_ = i // 避免未使用变量警告
	}

	// 处理对象别名
	switch object {
	case "a", "add", "addr", "address":
		return builtin.ShowIPAddr()
	case "l", "link":
		return builtin.ShowInterfaces()
	case "r", "route":
		return builtin.ShowRoutes()
	case "help", "":
		fmt.Println(getIPHelpText())
		return nil
	default:
		return fmt.Errorf("对象 \"%s\" 未知，请尝试 \"ip help\"", object)
	}
}

// getIPHelpText 返回 ip 命令的帮助文本
func getIPHelpText() string {
	return `用法: ip [ OPTIONS ] OBJECT { COMMAND | help }
      ip [ -force ] -batch filename

OBJECT := { link | address | route | help }
OPTIONS := { -V[ersion] | -h[uman-readable] | -s[tatistics] |
             -r[esolve] | -f[amily] { inet | inet6 } |
             -4 | -6 | -o[neline] | -br[ief] }

常用命令:
  ip addr           显示所有网络接口的 IP 地址
  ip addr show      显示所有网络接口的 IP 地址
  ip link           显示所有网络接口信息
  ip link show      显示所有网络接口信息
  ip route          显示路由表
  ip route show     显示路由表

简写形式:
  ip a              等同于 ip addr
  ip l              等同于 ip link
  ip r              等同于 ip route`
}
