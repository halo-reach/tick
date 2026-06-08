package cli

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
)

func newCredentialCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "credential",
		Short: "凭证中心管理（bearer / basic / oauth 等）",
		Long: `管理凭证中心条目——HTTP 鉴权所需的可复用凭据。

子命令:
  list    列出凭证
  get     获取凭证详情
  create  创建凭证
  delete  删除凭证

示例:
  tick credential list
  tick credential get cred_xxx
  tick credential create --name "api-key-A" --type bearer --config '{"token":"tk_xxx"}'
  tick credential delete cred_xxx --yes`,
	}

	createCmd := &cobra.Command{
		Use:   "create",
		Short: "创建凭证",
		Long: `创建凭证条目。--config 是 JSON 字符串，依 --type 不同:
  - bearer: {"token":"tk_xxx"}
  - basic:  {"username":"...","password":"..."}
  - oauth2: {"client_id":"...","client_secret":"...","token_url":"..."}

示例:
  tick credential create --name "prod-api" --type bearer --config '{"token":"tk_xxx"}'
  tick credential create --name "basic-auth" --type basic --config '{"username":"u","password":"p"}'`,
		Run: func(cmd *cobra.Command, args []string) {
			name, _ := cmd.Flags().GetString("name")
			ctype, _ := cmd.Flags().GetString("type")
			config, _ := cmd.Flags().GetString("config")

			body := map[string]any{
				"name":   name,
				"type":   ctype,
				"config": json.RawMessage(config),
			}
			b, _ := json.Marshal(body)
			printResp(doPost(cmd, "/api/v1/credentials", string(b)))
		},
	}
	createCmd.Flags().String("name", "", "credential name (required)")
	createCmd.Flags().String("type", "bearer", "credential type: bearer|basic|oauth2")
	createCmd.Flags().String("config", "{}", "credential config JSON")

	listCmd := &cobra.Command{
		Use:   "list",
		Short: "列出凭证",
		Long: `列出凭证条目（不显示 config 完整值）。

示例:
  tick credential list
  tick credential list --output json`,
		Run: func(cmd *cobra.Command, args []string) {
			printResp(doGet(cmd, "/api/v1/credentials"))
		},
	}

	getCmd := &cobra.Command{
		Use:   "get <id>",
		Short: "获取凭证详情",
		Long: `获取凭证详情（含解密后的 config）。

示例:
  tick credential get cred_xxx`,
		Args: cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			printResp(doGet(cmd, "/api/v1/credentials/"+args[0]))
		},
	}

	deleteCmd := &cobra.Command{
		Use:   "delete <id>",
		Short: "删除凭证",
		Long: `删除凭证。--yes 跳过确认（危险操作；被 task 引用的凭证可能被拒绝）。

示例:
  tick credential delete cred_xxx --yes`,
		Args: cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			yes, _ := cmd.Flags().GetBool("yes")
			if !confirm(yes, fmt.Sprintf("确认删除凭证 %s? (此操作不可逆)", args[0])) {
				fmt.Println("已取消")
				return
			}
			printResp(doDelete(cmd, "/api/v1/credentials/"+args[0]))
		},
	}
	deleteCmd.Flags().BoolP("yes", "y", false, "skip confirmation")

	cmd.AddCommand(createCmd, listCmd, getCmd, deleteCmd)
	return cmd
}
