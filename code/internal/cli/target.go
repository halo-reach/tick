package cli

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
)

func newTargetCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "target",
		Short: "触发目标管理（HTTP endpoint 定义）",
		Long: `管理触发目标——可复用的 HTTP endpoint 定义。

子命令:
  list    列出目标
  get     获取目标详情
  create  创建目标
  delete  删除目标

示例:
  tick target list
  tick target get tgt_xxx
  tick target create --name "webhook-a" --type http --url https://example.com/hook
  tick target delete tgt_xxx --yes`,
	}

	createCmd := &cobra.Command{
		Use:   "create",
		Short: "创建触发目标",
		Long: `创建触发目标（HTTP endpoint）。可被 task 复用。

示例:
  tick target create --name "webhook-a" --type http --url https://example.com/hook
  tick target create --name "slack-notify" --type http --url https://hooks.slack.com/xxx --method POST`,
		Run: func(cmd *cobra.Command, args []string) {
			name, _ := cmd.Flags().GetString("name")
			ttype, _ := cmd.Flags().GetString("type")
			url, _ := cmd.Flags().GetString("url")
			method, _ := cmd.Flags().GetString("method")

			config := map[string]any{"url": url, "method": method}
			configJSON, _ := json.Marshal(config)
			body := map[string]any{
				"name":   name,
				"type":   ttype,
				"config": json.RawMessage(configJSON),
			}
			b, _ := json.Marshal(body)
			printResp(doPost(cmd, "/api/v1/targets", string(b)))
		},
	}
	createCmd.Flags().String("name", "", "target name (required)")
	createCmd.Flags().String("type", "http", "target type")
	createCmd.Flags().String("url", "", "HTTP URL")
	createCmd.Flags().String("method", "POST", "HTTP method")

	listCmd := &cobra.Command{
		Use:   "list",
		Short: "列出触发目标",
		Long: `列出所有触发目标。

示例:
  tick target list
  tick target list --output json`,
		Run: func(cmd *cobra.Command, args []string) {
			printResp(doGet(cmd, "/api/v1/targets"))
		},
	}

	getCmd := &cobra.Command{
		Use:   "get <id>",
		Short: "获取触发目标详情",
		Long: `获取触发目标详情。

示例:
  tick target get tgt_xxx`,
		Args: cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			printResp(doGet(cmd, "/api/v1/targets/"+args[0]))
		},
	}

	deleteCmd := &cobra.Command{
		Use:   "delete <id>",
		Short: "删除触发目标",
		Long: `删除触发目标。--yes 跳过确认（危险操作）。

示例:
  tick target delete tgt_xxx --yes`,
		Args: cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			yes, _ := cmd.Flags().GetBool("yes")
			if !confirm(yes, fmt.Sprintf("确认删除目标 %s? (此操作不可逆)", args[0])) {
				fmt.Println("已取消")
				return
			}
			printResp(doDelete(cmd, "/api/v1/targets/"+args[0]))
		},
	}
	deleteCmd.Flags().BoolP("yes", "y", false, "skip confirmation")

	cmd.AddCommand(createCmd, listCmd, getCmd, deleteCmd)
	return cmd
}
