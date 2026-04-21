package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"opsxcli/plugins/builtin"
)

func init() {
	RegisterCommand("dd", "系统", "磁盘读写", NewDdCmd)
}

// === 磁盘工具 ===

func NewDdCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "dd [options]",
		Short: "转换和复制文件",
		Long: `转换和复制文件,支持底层数据复制

参数格式: key=value

常用选项:
  if=FILE         输入文件 (默认: stdin)
  of=FILE         输出文件 (默认: stdout)
  bs=BYTES        块大小 (默认: 512)
  count=N         复制 N 个块
  skip=N          跳过输入文件的前 N 个块
  seek=N          跳过输出文件的前 N 个块
  conv=CONVS      转换选项 (notrunc: 不截断输出文件)
  status=LEVEL    显示级别 (progress, noxfer, none)

大小单位: K (1024), M (1024*1024), G (1024*1024*1024)

示例:
  # 创建 100MB 的空文件
  opsxcli dd if=/dev/zero of=test.img bs=1M count=100

  # 复制文件
  opsxcli dd if=input.bin of=output.bin bs=4K

  # 备份磁盘分区
  opsxcli dd if=/dev/sda1 of=backup.img bs=1M status=progress

  # 创建引导盘
  opsxcli dd if=ubuntu.iso of=/dev/sdb bs=4M status=progress

  # 擦除磁盘数据
  opsxcli dd if=/dev/zero of=/dev/sdb bs=1M count=1024

  # 从文件中读取特定位置的数据
  opsxcli dd if=data.bin of=output.bin bs=512 skip=10 count=20`,
		DisableFlagParsing: true, // 禁用标准参数解析,使用自定义格式
		RunE: func(c *cobra.Command, args []string) error {
			// 处理帮助请求
			if len(args) > 0 && (args[0] == "-h" || args[0] == "--help" || args[0] == "help") {
				fmt.Println(c.Long)
				return nil
			}

			opts, err := builtin.ParseDdArgs(args)
			if err != nil {
				return fmt.Errorf("参数错误: %v", err)
			}
			return builtin.Dd(opts)
		},
	}
	return cmd
}
