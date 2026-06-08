package cli

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

func newTaskCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "task",
		Short: "任务管理（CRUD + 调度）",
		Long: `管理定时任务——CRUD、调度参数、启停、删除与历史。

子命令:
  list     列出任务（支持 status 过滤）
  get      获取任务详情
  create   创建任务（cron/every/at）
  pause    暂停任务
  resume   恢复任务
  delete   删除任务
  history  查看执行历史

示例:
  tick task list
  tick task list --status active --output json
  tick task get t_abc123
  tick task create --name "每日同步" --cron "0 8 * * *" --url https://example.com/hook
  tick task create --name "每 5 分钟" --every 5m --url https://example.com/hook
  tick task pause t_abc123
  tick task resume t_abc123
  tick task delete t_abc123 --yes
  tick task history t_abc123 --limit 50`,
	}

	createCmd := &cobra.Command{
		Use:   "create",
		Short: "创建一个定时任务",
		Long: `创建一个定时任务。--cron / --every / --at 三选一。
  - --cron  "0 8 * * *"           6 字段 cron 表达式
  - --every 5m                   固定间隔（30s / 5m / 1h / ...）
  - --at    2026-06-02T20:00:00Z  一次性触发

示例:
  tick task create --name "日报" --cron "0 8 * * *" --url https://example.com/hook
  tick task create --name "心跳" --every 30s --url https://example.com/hook --method GET`,
		Run: func(cmd *cobra.Command, args []string) {
			name, _ := cmd.Flags().GetString("name")
			cronExpr, _ := cmd.Flags().GetString("cron")
			every, _ := cmd.Flags().GetString("every")
			at, _ := cmd.Flags().GetString("at")
			url, _ := cmd.Flags().GetString("url")
			method, _ := cmd.Flags().GetString("method")
			headers, _ := cmd.Flags().GetString("headers")
			bodyData, _ := cmd.Flags().GetString("body")
			targetID, _ := cmd.Flags().GetString("target-id")
			timeout, _ := cmd.Flags().GetInt("timeout")
			retry, _ := cmd.Flags().GetInt("retry")

			scheduleType, schedParam, err := resolveSchedule(cronExpr, every, at)
			if err != nil {
				exitErr("tick task create", err)
			}
			body := map[string]any{
				"name":          name,
				"schedule_type": scheduleType,
				"timeout_secs":  timeout,
				"retry_count":   retry,
			}
			if schedParam != "" {
				body[scheduleType+"_expr"] = schedParam
				if scheduleType == "cron" {
					body["cron_expr"] = schedParam
				}
				if scheduleType == "interval" {
					body["interval_secs"] = parseDurationSeconds(schedParam)
				}
				if scheduleType == "once" {
					body["at_time"] = schedParam
				}
			}
			if targetID != "" {
				body["target_id"] = targetID
			} else if url != "" {
				body["url"] = url
				body["method"] = method
				if strings.TrimSpace(headers) != "" {
					body["headers"] = json.RawMessage(headers)
				}
				if strings.TrimSpace(bodyData) != "" {
					body["body"] = json.RawMessage(bodyData)
				}
			}
			b, _ := json.Marshal(body)
			printResp(doPost(cmd, "/api/v1/tasks", string(b)))
		},
	}
	createCmd.Flags().String("name", "", "task name (required)")
	createCmd.Flags().String("cron", "", "6-field cron expression")
	createCmd.Flags().String("every", "", "fixed interval (e.g. 30s, 5m, 1h)")
	createCmd.Flags().String("at", "", "one-shot trigger (RFC3339)")
	createCmd.Flags().String("url", "", "target URL (creates inline target)")
	createCmd.Flags().String("method", "POST", "HTTP method")
	createCmd.Flags().String("headers", "", "HTTP headers JSON")
	createCmd.Flags().String("body", "", "HTTP body JSON")
	createCmd.Flags().String("target-id", "", "reuse existing target id")
	createCmd.Flags().Int("timeout", 30, "timeout seconds")
	createCmd.Flags().Int("retry", 3, "max retries")

	listCmd := &cobra.Command{
		Use:   "list",
		Short: "列出任务",
		Long: `列出任务（支持按 status 过滤）。

示例:
  tick task list
  tick task list --status active
  tick task list --output json`,
		Run: func(cmd *cobra.Command, args []string) {
			status, _ := cmd.Flags().GetString("status")
			path := "/api/v1/tasks"
			if status != "" {
				path += "?status=" + status
			}
			printResp(doGet(cmd, path))
		},
	}
	listCmd.Flags().String("status", "", "filter by status: active|paused|deleted")

	getCmd := &cobra.Command{
		Use:   "get <id>",
		Short: "获取任务详情",
		Long: `获取任务详情。

示例:
  tick task get t_abc123
  tick task get t_abc123 --output json`,
		Args: cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			printResp(doGet(cmd, "/api/v1/tasks/"+args[0]))
		},
	}

	pauseCmd := &cobra.Command{
		Use:   "pause <id>",
		Short: "暂停任务",
		Long: `暂停任务。--yes 跳过确认。

示例:
  tick task pause t_abc123
  tick task pause t_abc123 --yes`,
		Args: cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			yes, _ := cmd.Flags().GetBool("yes")
			if !confirm(yes, fmt.Sprintf("确认暂停任务 %s?", args[0])) {
				fmt.Println("已取消")
				return
			}
			printResp(doPost(cmd, "/api/v1/tasks/"+args[0]+"/pause", ""))
		},
	}
	pauseCmd.Flags().BoolP("yes", "y", false, "skip confirmation")

	resumeCmd := &cobra.Command{
		Use:   "resume <id>",
		Short: "恢复任务",
		Long: `恢复已暂停任务。

示例:
  tick task resume t_abc123`,
		Args: cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			printResp(doPost(cmd, "/api/v1/tasks/"+args[0]+"/resume", ""))
		},
	}

	deleteCmd := &cobra.Command{
		Use:   "delete <id>",
		Short: "删除任务",
		Long: `删除任务。--yes 跳过确认（危险操作）。

示例:
  tick task delete t_abc123 --yes`,
		Args: cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			yes, _ := cmd.Flags().GetBool("yes")
			if !confirm(yes, fmt.Sprintf("确认删除任务 %s? (此操作不可逆)", args[0])) {
				fmt.Println("已取消")
				return
			}
			printResp(doDelete(cmd, "/api/v1/tasks/"+args[0]))
		},
	}
	deleteCmd.Flags().BoolP("yes", "y", false, "skip confirmation")

	historyCmd := &cobra.Command{
		Use:   "history <id>",
		Short: "查看执行历史",
		Long: `查看任务的执行历史记录。

示例:
  tick task history t_abc123
  tick task history t_abc123 --limit 50 --output json`,
		Args: cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			limit, _ := cmd.Flags().GetInt("limit")
			printResp(doGet(cmd, fmt.Sprintf("/api/v1/tasks/%s/history?limit=%d", args[0], limit)))
		},
	}
	historyCmd.Flags().Int("limit", 20, "max number of history records")

	cmd.AddCommand(createCmd, listCmd, getCmd, pauseCmd, resumeCmd, deleteCmd, historyCmd)
	return cmd
}

func resolveSchedule(cronExpr, every, at string) (string, string, error) {
	count := 0
	if cronExpr != "" {
		count++
	}
	if every != "" {
		count++
	}
	if at != "" {
		count++
	}
	if count != 1 {
		return "", "", fmt.Errorf("调度参数必须三选一 (--cron/--every/--at)，当前 %d 个", count)
	}
	if cronExpr != "" {
		return "cron", cronExpr, nil
	}
	if every != "" {
		return "interval", every, nil
	}
	return "once", at, nil
}

func parseDurationSeconds(s string) int {
	if d, err := time.ParseDuration(s); err == nil {
		return int(d.Seconds())
	}
	return 0
}
