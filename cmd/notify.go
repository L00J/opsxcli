package cmd

import (
	"fmt"
	"strings"

	"opsxcli/plugins/notify"

	"github.com/spf13/cobra"
)

func init() {
	RegisterCommand("notify", "工具", "告警通知工具", NewNotifyCmd)
}

// NewNotifyCmd creates the notify command.
func NewNotifyCmd() *cobra.Command {
	var (
		target string
		nType  string
		title  string
		level  string
		at     []string
		secret string
	)

	cmd := &cobra.Command{
		Use:   "notify <message>",
		Short: "发送告警通知",
		Long: `发送告警通知到指定目标（飞书、钉钉、通用 Webhook）。

支持多种通知渠道和消息级别，可通过参数指定目标地址和类型。

示例：
  opsxcli notify "部署完成" --target https://hook.example.com --type webhook
  opsxcli notify "服务器异常" --target https://open.feishu.cn/open-apis/bot/v2/hook/xxx --type feishu --level error
  opsxcli notify "CPU 告警" --target https://oapi.dingtalk.com/robot/send?access_token=xxx --type dingtalk --level critical --at 13800138000`,
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			message := strings.Join(args, " ")

			// Validate level
			if !notify.IsValidLevel(level) {
				return fmt.Errorf("invalid level %q, must be one of: %s", level, strings.Join(notify.ValidLevels(), ", "))
			}

			// Validate target
			if target == "" {
				return fmt.Errorf("target is required, use --target to specify the webhook URL")
			}

			// Create notifier
			notifier, err := notify.NewNotifier(nType, target)
			if err != nil {
				return fmt.Errorf("failed to create notifier: %w", err)
			}

			// Set secret for sign verification if applicable
			type secretSetter interface {
				SetSecret(string)
			}
			if secret != "" {
				if setter, ok := notifier.(secretSetter); ok {
					setter.SetSecret(secret)
				}
			}

			// Build message
			msg := &notify.Message{
				Title:   title,
				Content: message,
				Level:   level,
				AtUsers: at,
			}

			fmt.Fprintf(cmd.OutOrStdout(), "发送通知 [%s] 到 %s ...\n", notifier.Name(), target)

			if err := notifier.Send(msg); err != nil {
				return fmt.Errorf("failed to send notification: %w", err)
			}

			fmt.Fprintf(cmd.OutOrStdout(), "通知发送成功!\n")
			return nil
		},
	}

	cmd.Flags().StringVarP(&target, "target", "t", "", "通知目标 URL")
	cmd.Flags().StringVar(&nType, "type", "webhook", "通知类型 (webhook/feishu/dingtalk)")
	cmd.Flags().StringVar(&title, "title", "OpsXCLI 通知", "消息标题")
	cmd.Flags().StringVarP(&level, "level", "l", "info", "告警级别 (info/warning/error/critical)")
	cmd.Flags().StringSliceVar(&at, "at", nil, "@用户 (逗号分隔)")
	cmd.Flags().StringVar(&secret, "secret", "", "签名密钥")

	return cmd
}
