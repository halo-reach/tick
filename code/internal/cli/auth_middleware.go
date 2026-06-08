package cli

import (
	"context"
	"strings"

	"github.com/spf13/cobra"
)

// apiKeyContextKey is unexported to prevent collisions with keys from
// other packages (https://pkg.go.dev/context#WithValue).
type apiKeyContextKey struct{}

// authExemptTestPrefix is recognized by isAuthExempt so unit tests can
// inject synthetic commands without leaking them to the real CLI.
const authExemptTestPrefix = "__test_"

// isAuthExempt reports whether cmd may run without an authenticated API key.
// The exemption is based on (parent, name) so that e.g. "auth login" is
// exempt but "task login" (if it existed) would not be.
func isAuthExempt(cmd *cobra.Command) bool {
	if cmd == nil {
		return true
	}
	name := cmd.Name()
	if name == "help" || name == "version" || name == "completion" {
		return true
	}
	if name == "update" {
		return true
	}
	if strings.HasPrefix(name, authExemptTestPrefix) {
		return true
	}
	if cmd.Parent() != nil {
		parent := cmd.Parent().Name()
		if parent == "auth" && (name == "login" || name == "logout" || name == "whoami") {
			return true
		}
	}
	return false
}

// RequireAuth is a cobra PersistentPreRunE hook. It runs before every
// non-exempt subcommand, resolves the API key (env > config), and injects
// it into cmd.Context() for downstream consumers (notably doRequest).
// On failure it routes through handleCLIError (so JSON mode produces JSON)
// and exits with code 2 (FR-035, FR-036, FR-039, FR-043).
func RequireAuth(cmd *cobra.Command, args []string) error {
	if isAuthExempt(cmd) {
		return nil
	}
	cfg, _ := LoadConfig()
	key := ResolveAPIKey(cfg)
	if key == "" {
		cmd.SilenceUsage = true
		cmd.SilenceErrors = true
		handleCLIError(cmd, &CLIError{
			Reason:        "未登录",
			ServerMessage: "未找到可用的 API key",
			FixSuggestion: "请先执行 tick auth login",
			ExitCode:      ExitNotLogged,
		})
	}
	ctx := cmd.Context()
	if ctx == nil {
		ctx = context.Background()
	}
	cmd.SetContext(context.WithValue(ctx, apiKeyContextKey{}, key))
	return nil
}

// APIKeyFromContext returns the API key injected by RequireAuth. Returns ""
// when called outside a command tree (e.g. direct unit tests) or when the
// key was never injected.
func APIKeyFromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	if v, ok := ctx.Value(apiKeyContextKey{}).(string); ok {
		return v
	}
	return ""
}
