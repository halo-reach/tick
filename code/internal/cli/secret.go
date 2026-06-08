package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newSecretCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "secret",
		Short: "签名密钥管理（用于请求签名验证）",
		Long: `管理签名密钥——用于回调请求签名验证。

子命令:
  list    列出签名密钥
  create  创建新签名密钥
  revoke  撤销签名密钥

示例:
  tick secret list
  tick secret create
  tick secret revoke sec_xxx --yes`,
	}

	listCmd := &cobra.Command{
		Use:   "list",
		Short: "列出签名密钥",
		Long: `列出签名密钥。

示例:
  tick secret list
  tick secret list --output json`,
		Run: func(cmd *cobra.Command, args []string) {
			printResp(doGet(cmd, "/api/v1/secrets"))
		},
	}

	createCmd := &cobra.Command{
		Use:   "create",
		Short: "创建新签名密钥",
		Long: `创建新的签名密钥。返回值包含 secret value（仅此一次可见）。

示例:
  tick secret create`,
		Run: func(cmd *cobra.Command, args []string) {
			printResp(doPost(cmd, "/api/v1/secrets", ""))
		},
	}

	revokeCmd := &cobra.Command{
		Use:   "revoke <id>",
		Short: "撤销签名密钥",
		Long: `撤销签名密钥。--yes 跳过确认（危险操作）。

示例:
  tick secret revoke sec_xxx --yes`,
		Args: cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			yes, _ := cmd.Flags().GetBool("yes")
			if !confirm(yes, fmt.Sprintf("确认撤销密钥 %s? (此操作不可逆)", args[0])) {
				fmt.Println("已取消")
				return
			}
			printResp(doDelete(cmd, "/api/v1/secrets/"+args[0]))
		},
	}
	revokeCmd.Flags().BoolP("yes", "y", false, "skip confirmation")

	cmd.AddCommand(listCmd, createCmd, revokeCmd)
	return cmd
}
