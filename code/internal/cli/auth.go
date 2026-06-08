package cli

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

func newAuthCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "auth",
		Short: "认证与租户管理",
		Long: `auth 子命令管理 API key 与租户身份。

子命令:
  login   设置当前 BuiltForEnv 对应的 API key
  logout  清除当前 BuiltForEnv 对应的 API key
  whoami  显示 tenant / api_key 前缀 / server URL + BuiltForEnv / built
  keys    API key 管理（list / create / revoke）
  quota   显示租户配额
  status  显示租户状态

示例:
  tick auth login --api-key tk_xxx
  tick auth logout
  tick whoami`,
	}

	loginCmd := &cobra.Command{
		Use:   "login",
		Short: "设置当前 BuiltForEnv 对应的 API key（URL 已硬编码进二进制）",
		Long: `设置当前二进制对应的 API key，URL 来自编译期常量，无需手动指定。

示例:
  tick auth login --api-key tk_xxx
  tick auth login         # 交互式输入`,
		Run: func(cmd *cobra.Command, args []string) {
			key, _ := cmd.Flags().GetString("api-key")
			if key == "" {
				fmt.Fprintf(os.Stderr, "当前二进制连的是 %s (%s)，请输入 API key: ", ServerURL(), BuiltForEnv())
				reader := bufio.NewReader(os.Stdin)
				line, _ := reader.ReadString('\n')
				key = strings.TrimSpace(line)
			}
			if key == "" {
				exitErr("API key 不能为空", nil)
			}
			cfg, err := LoadConfig()
			if err != nil {
				exitErr("读取配置失败", err)
			}
			cfg.APIKeys[BuiltForEnv()] = key
			if err := SaveConfig(cfg); err != nil {
				exitErr("保存配置失败", err)
			}
			fmt.Printf("登录成功 (env=%s)\n", BuiltForEnv())
		},
	}
	loginCmd.Flags().String("api-key", "", "API key (non-interactive)")

	logoutCmd := &cobra.Command{
		Use:   "logout",
		Short: "清除当前 BuiltForEnv 对应的 API key（其他 env 不动）",
		Long: `清除当前 BuiltForEnv 对应槽位的 API key，其他 env 的 key 不受影响。

示例:
  tick auth logout
  tick auth logout --yes   # 跳过确认`,
		Run: func(cmd *cobra.Command, args []string) {
			yes, _ := cmd.Flags().GetBool("yes")
			cfg, err := LoadConfig()
			if err != nil {
				exitErr("读取配置失败", err)
			}
			if _, ok := cfg.APIKeys[BuiltForEnv()]; !ok || cfg.APIKeys[BuiltForEnv()] == "" {
				fmt.Println("无操作（未登录）")
				return
			}
			if !yes {
				fmt.Fprintf(os.Stderr, "确认登出 %s 环境的 API key? [y/N]: ", BuiltForEnv())
				reader := bufio.NewReader(os.Stdin)
				line, _ := reader.ReadString('\n')
				ans := strings.ToLower(strings.TrimSpace(line))
				if ans != "y" && ans != "yes" {
					fmt.Println("已取消")
					return
				}
			}
			delete(cfg.APIKeys, BuiltForEnv())
			if err := SaveConfig(cfg); err != nil {
				exitErr("保存配置失败", err)
			}
			fmt.Printf("已登出 (env=%s)\n", BuiltForEnv())
		},
	}
	logoutCmd.Flags().BoolP("yes", "y", false, "skip confirmation")

	whoamiCmd := &cobra.Command{
		Use:   "whoami",
		Short: "显示 tenant / api_key 前缀 / server URL + BuiltForEnv / built 四元组",
		Long: `显示当前身份与连接的环境，一眼区分 prod / SIT:
  tenant:     <tenant>
  user:       <user>
  api_key:    <prefix>...<suffix>
  server:     <server URL> (<BuiltForEnv>)
  built:      <version> (commit <sha>, built <time>)

示例:
  tick whoami
  tick whoami --output json`,
		Run: func(cmd *cobra.Command, args []string) {
			k := APIKeyFromContext(cmd.Context())
			if k == "" {
				if cfg, _ := LoadConfig(); cfg != nil {
					k = cfg.APIKeys[BuiltForEnv()]
				}
			}
			prefix := ""
			if len(k) > 12 {
				prefix = k[:8] + "..." + k[len(k)-6:]
			} else if k != "" {
				prefix = k
			}
			tenant, user := fetchWhoamiIdentity(cmd)
			if outputFmt == "json" {
				out := map[string]any{
					"tenant":         tenant,
					"user":           user,
					"api_key_prefix": prefix,
					"server_url":     ServerURL(),
					"built_for_env":  BuiltForEnv(),
					"version":        buildVersion,
					"commit":         buildCommit,
					"build_time":     buildTime,
				}
				b, _ := json.MarshalIndent(out, "", "  ")
				fmt.Println(string(b))
				return
			}
			fmt.Printf("tenant:     %s\n", tenant)
			fmt.Printf("user:       %s\n", user)
			fmt.Printf("api_key:    %s\n", prefix)
			fmt.Printf("server:     %s (%s)\n", ServerURL(), BuiltForEnv())
			fmt.Printf("built:      %s (commit %s, %s)\n", buildVersion, buildCommit, buildTime)
		},
	}

	keysCmd := &cobra.Command{Use: "keys", Short: "API key 管理"}
	keysCmd.AddCommand(
		&cobra.Command{Use: "list", Short: "列出 API key", Run: func(cmd *cobra.Command, args []string) {
			printResp(doGet(cmd, "/api/v1/auth/keys"))
		}},
		&cobra.Command{Use: "create", Short: "创建新 API key", Run: func(cmd *cobra.Command, args []string) {
			printResp(doPost(cmd, "/api/v1/auth/keys", ""))
		}},
		&cobra.Command{Use: "revoke", Short: "撤销 API key", Args: cobra.ExactArgs(1), Run: func(cmd *cobra.Command, args []string) {
			printResp(doDelete(cmd, "/api/v1/auth/keys/"+args[0]))
		}},
	)

	quotaCmd := &cobra.Command{
		Use:   "quota",
		Short: "显示租户配额",
		Run: func(cmd *cobra.Command, args []string) {
			printResp(doGet(cmd, "/api/v1/quota"))
		},
	}

	statusCmd := &cobra.Command{
		Use:   "status",
		Short: "显示租户状态",
		Run: func(cmd *cobra.Command, args []string) {
			printResp(doGet(cmd, "/api/v1/status"))
		},
	}

	cmd.AddCommand(loginCmd, logoutCmd, whoamiCmd, keysCmd, quotaCmd, statusCmd)
	return cmd
}

// fetchWhoamiIdentity pulls tenant/user info from the server. Returns "" on error
// or when the API key is missing — whoami is exempt from RequireAuth and must
// still print local metadata when the user is not logged in.
func fetchWhoamiIdentity(cmd *cobra.Command) (tenant, user string) {
	k := APIKeyFromContext(cmd.Context())
	if k == "" {
		return "", ""
	}
	body := doGet(cmd, "/api/v1/auth/keys")
	var m map[string]any
	if err := json.Unmarshal(body, &m); err != nil {
		return "", ""
	}
	if t, ok := m["tenant"].(string); ok {
		tenant = t
	}
	if u, ok := m["user"].(string); ok {
		user = u
	}
	return tenant, user
}

func doGet(cmd *cobra.Command, path string) []byte {
	return doRequest(cmd, "GET", path, "")
}

func doPost(cmd *cobra.Command, path, body string) []byte {
	return doRequest(cmd, "POST", path, body)
}

func doDelete(cmd *cobra.Command, path string) []byte {
	return doRequest(cmd, "DELETE", path, "")
}

func doPut(cmd *cobra.Command, path, body string) []byte {
	return doRequest(cmd, "PUT", path, body)
}

// doRequest executes an HTTP call to ServerURL() + path. The API key is
// read from cmd.Context() (injected by RequireAuth). Server URL is always
// ServerURL() (compile-time, no runtime override). All error paths flow
// through MapHTTPError / MapNetworkError and handleCLIError — no inline
// error formatting in this function (FR-040, FR-043, SC-014).
func doRequest(cmd *cobra.Command, method, path, body string) []byte {
	k := APIKeyFromContext(cmd.Context())
	if k == "" {
		// Defensive fallback for tests / direct calls that bypass RequireAuth.
		if cfg, _ := LoadConfig(); cfg != nil {
			k = cfg.APIKeys[BuiltForEnv()]
		}
	}
	if k == "" {
		handleCLIError(cmd, &CLIError{
			Reason:        "NotLoggedIn",
			ServerMessage: "未找到可用的 API key",
			FixSuggestion: "请先执行 tick auth login，或设置 TICK_API_KEY 环境变量",
			ExitCode:      ExitNotLogged,
		})
	}
	var bodyReader io.Reader
	if body != "" {
		bodyReader = strings.NewReader(body)
	}
	req, err := http.NewRequest(method, ServerURL()+path, bodyReader)
	if err != nil {
		handleCLIError(cmd, MapNetworkError(err))
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+k)
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		handleCLIError(cmd, MapNetworkError(err))
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		retryAfter := resp.Header.Get("Retry-After")
		handleCLIError(cmd, MapHTTPError(resp.StatusCode, bytes.TrimSpace(b), retryAfter))
	}
	return b
}

func printResp(data []byte) {
	if outputFmt == "json" {
		fmt.Println(string(data))
		return
	}
	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		fmt.Println(string(data))
		return
	}
	if arr, ok := m["data"].([]any); ok {
		m["data"] = arr
		m["total"] = len(arr)
	}
	b, _ := json.MarshalIndent(m, "", "  ")
	fmt.Println(string(b))
}

// ensureDir creates ~/.tick with 0700 if missing.
func ensureDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(home, ".tick")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", err
	}
	return dir, nil
}
