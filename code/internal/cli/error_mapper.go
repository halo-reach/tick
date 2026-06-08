package cli

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
)

const (
	ExitOK         = 0
	ExitBusiness   = 1
	ExitNotLogged  = 2
	ExitUnavailable = 3
	ExitRateLimit  = 4
)

type CLIError struct {
	StatusCode    int    `json:"status_code"`
	Reason        string `json:"reason"`
	ServerMessage string `json:"server_message"`
	FixSuggestion string `json:"fix_suggestion"`
	ExitCode      int    `json:"exit_code"`
}

func (e *CLIError) Error() string {
	if e == nil {
		return ""
	}
	if e.StatusCode == 0 {
		return fmt.Sprintf("%s: %s", e.Reason, e.ServerMessage)
	}
	return fmt.Sprintf("%d %s: %s", e.StatusCode, e.Reason, e.ServerMessage)
}

func ExitCodeFor(statusCode int, isNetwork bool) int {
	if isNetwork {
		return ExitUnavailable
	}
	switch statusCode {
	case http.StatusUnauthorized:
		return ExitNotLogged
	case http.StatusTooManyRequests:
		return ExitRateLimit
	case http.StatusInternalServerError,
		http.StatusBadGateway,
		http.StatusServiceUnavailable,
		http.StatusGatewayTimeout:
		return ExitUnavailable
	}
	if statusCode >= 500 {
		return ExitUnavailable
	}
	return ExitBusiness
}

func MapHTTPError(statusCode int, body []byte, retryAfter string) *CLIError {
	serverMsg := strings.TrimSpace(string(body))
	fix := "请求问题，参考 tick --help"
	exit := ExitBusiness
	switch statusCode {
	case http.StatusBadRequest:
		fix = "检查参数"
		exit = ExitBusiness
	case http.StatusUnauthorized:
		fix = "API key 无效或过期，请重新执行 tick auth login"
		exit = ExitNotLogged
	case http.StatusForbidden:
		fix = "权限不足，联系租户 Owner"
		exit = ExitBusiness
	case http.StatusNotFound:
		fix = "资源不存在（ID 是否正确？）"
		exit = ExitBusiness
	case http.StatusConflict:
		fix = "资源冲突（重名 / 已删除 / 已存在）"
		exit = ExitBusiness
	case http.StatusUnprocessableEntity:
		fix = "参数语义错误（cron 表达式、URL 格式）"
		exit = ExitBusiness
	case http.StatusTooManyRequests:
		secs := parseRetryAfter(retryAfter)
		fix = fmt.Sprintf("请求过于频繁，Retry-After: %d 秒（可指数退避重试）", secs)
		exit = ExitRateLimit
	case http.StatusInternalServerError:
		fix = "服务端内部错误，稍后重试或联系管理员"
		exit = ExitUnavailable
	case http.StatusBadGateway, http.StatusServiceUnavailable, http.StatusGatewayTimeout:
		fix = "上游网关/服务不可用，稍后重试"
		exit = ExitUnavailable
	default:
		if statusCode >= 500 {
			fix = "服务端问题，稍后重试"
			exit = ExitUnavailable
		} else {
			fix = "请求问题，参考 tick --help"
			exit = ExitBusiness
		}
	}
	return &CLIError{
		StatusCode:    statusCode,
		Reason:        http.StatusText(statusCode),
		ServerMessage: serverMsg,
		FixSuggestion: fix,
		ExitCode:      exit,
	}
}

func MapNetworkError(err error) *CLIError {
	if err == nil {
		return nil
	}
	reason := "NetworkError"
	switch e := err.(type) {
	case *url.Error:
		reason = "URLRequestError"
	case net.Error:
		if e.Timeout() {
			reason = "NetworkTimeout"
		} else {
			reason = "NetworkError"
		}
	}
	return &CLIError{
		StatusCode:    0,
		Reason:        reason,
		ServerMessage: err.Error(),
		FixSuggestion: "检查网络 / 服务可达性 / 代理设置",
		ExitCode:      ExitUnavailable,
	}
}

// parseRetryAfter handles both numeric seconds and HTTP-date formats.
// Returns 0 when the header is empty or unparseable.
func parseRetryAfter(s string) int {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0
	}
	if n, err := strconv.Atoi(s); err == nil {
		if n < 0 {
			return 0
		}
		return n
	}
	return 0
}

func handleCLIError(cmd *cobra.Command, err *CLIError) {
	if err == nil {
		return
	}
	if outputFmt == "json" {
		b, mErr := json.Marshal(err)
		if mErr != nil {
			fmt.Fprintf(os.Stderr, "internal: marshal CLIError failed: %v\n", mErr)
		} else {
			fmt.Fprintln(os.Stderr, string(b))
		}
	} else {
		if err.StatusCode == 0 {
			fmt.Fprintf(os.Stderr, "tick: %s: %s (%s)\n",
				err.Reason, err.ServerMessage, err.FixSuggestion)
		} else {
			fmt.Fprintf(os.Stderr, "tick %s: %d %s: %s (%s)\n",
				cmdPath(cmd), err.StatusCode, err.Reason, err.ServerMessage, err.FixSuggestion)
		}
	}
	os.Exit(err.ExitCode)
}

// cmdPath reconstructs a human-readable "tick <subcmd>" path from the cobra
// command tree, e.g. "task list" or "auth whoami". Falls back to cmd.Name().
func cmdPath(cmd *cobra.Command) string {
	if cmd == nil {
		return ""
	}
	parts := []string{}
	for c := cmd; c != nil; c = c.Parent() {
		if c.Name() == "" || c.Name() == "tick" {
			break
		}
		parts = append([]string{c.Name()}, parts...)
	}
	return strings.Join(parts, " ")
}
