package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	cfgFile   string
	outputFmt string

	// apiKey is resolved at request time, NOT bound to viper (FR-015).
	// Keeping it as a separate package variable prevents viper from ever
	// round-tripping TICK_API_KEY back to disk.
	apiKey string
)

func NewRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:   "tick",
		Short: "Tick - 秒级定时触发平台 CLI",
		Long: `Tick CLI — 任务/目标/凭证/密钥的日常管理与自更新。

快速开始:
  # 1. 登录（URL 已硬编码进二进制）
  tick auth login --api-key tk_xxx

  # 2. 查看身份
  tick whoami

  # 3. 日常使用
  tick task list
  tick task create --name foo --cron "0 8 * * *" --url https://example.com/hook

  # 4. 自更新
  tick update --check
  tick update

更多信息: https://github.com/tickplatform/tick`,
		Version:          VersionString(),
		PersistentPreRunE: RequireAuth,
	}

	root.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default ~/.tick/config.yaml)")
	root.PersistentFlags().StringVar(&outputFmt, "output", "table", "output format: table|json")

	root.AddCommand(newAuthCmd())
	root.AddCommand(newTaskCmd())
	root.AddCommand(newTargetCmd())
	root.AddCommand(newSecretCmd())
	root.AddCommand(newCredentialCmd())
	root.AddCommand(newUpdateCmd())

	root.SetVersionTemplate(VersionString() + "\n")

	cobra.OnInitialize(initConfig)
	return root
}

func initConfig() {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		home, _ := os.UserHomeDir()
		viper.AddConfigPath(filepath.Join(home, ".tick"))
		viper.SetConfigName("config")
		viper.SetConfigType("yaml")
	}
	// viper is only used here for legacy migration detection in LoadConfig;
	// apiKey is resolved separately and never goes through viper (FR-015).
	_ = viper.ReadInConfig()
}

// exitErr writes a formatted error to stderr and exits with code 1.
func exitErr(msg string, err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: %v\n", msg, err)
	} else {
		fmt.Fprintf(os.Stderr, "%s\n", msg)
	}
	os.Exit(1)
}
