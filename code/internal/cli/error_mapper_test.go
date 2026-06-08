package cli

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestMapHTTPError_AllStatusCodes(t *testing.T) {
	cases := []struct {
		code      int
		wantExit  int
		wantFixKW string
	}{
		{http.StatusBadRequest, ExitBusiness, "检查参数"},
		{http.StatusUnauthorized, ExitNotLogged, "tick auth login"},
		{http.StatusForbidden, ExitBusiness, "权限不足"},
		{http.StatusNotFound, ExitBusiness, "资源不存在"},
		{http.StatusConflict, ExitBusiness, "资源冲突"},
		{http.StatusUnprocessableEntity, ExitBusiness, "参数语义错误"},
		{http.StatusTooManyRequests, ExitRateLimit, "Retry-After"},
		{http.StatusInternalServerError, ExitUnavailable, "服务端内部错误"},
		{http.StatusBadGateway, ExitUnavailable, "上游"},
		{http.StatusServiceUnavailable, ExitUnavailable, "上游"},
		{http.StatusGatewayTimeout, ExitUnavailable, "上游"},
		{418, ExitBusiness, "请求问题"},
		{599, ExitUnavailable, "服务端问题"},
	}
	for _, c := range cases {
		name := fmt.Sprintf("%d_%s", c.code, http.StatusText(c.code))
		t.Run(name, func(t *testing.T) {
			got := MapHTTPError(c.code, []byte("server said: bad"), "30")
			if got.StatusCode != c.code {
				t.Errorf("StatusCode = %d, want %d", got.StatusCode, c.code)
			}
			if got.ExitCode != c.wantExit {
				t.Errorf("ExitCode = %d, want %d (status %d)", got.ExitCode, c.wantExit, c.code)
			}
			if !strings.Contains(got.FixSuggestion, c.wantFixKW) {
				t.Errorf("FixSuggestion %q missing keyword %q", got.FixSuggestion, c.wantFixKW)
			}
			if !strings.Contains(got.ServerMessage, "server said: bad") {
				t.Errorf("ServerMessage should preserve body, got %q", got.ServerMessage)
			}
		})
	}
}

func TestMapHTTPError_429_RetryAfterNumeric(t *testing.T) {
	got := MapHTTPError(http.StatusTooManyRequests, []byte("rate limit"), "30")
	if !strings.Contains(got.FixSuggestion, "30 秒") {
		t.Errorf("FixSuggestion should contain parsed 30 秒, got %q", got.FixSuggestion)
	}
}

func TestMapHTTPError_429_RetryAfterHTTPDate(t *testing.T) {
	got := MapHTTPError(http.StatusTooManyRequests, []byte("rate limit"), "Wed, 21 Oct 2015 07:28:00 GMT")
	if !strings.Contains(got.FixSuggestion, "Retry-After:") {
		t.Errorf("FixSuggestion should mention Retry-After, got %q", got.FixSuggestion)
	}
}

func TestMapHTTPError_429_EmptyRetryAfter(t *testing.T) {
	got := MapHTTPError(http.StatusTooManyRequests, []byte("rate limit"), "")
	if !strings.Contains(got.FixSuggestion, "0 秒") {
		t.Errorf("empty Retry-After should fall back to 0, got %q", got.FixSuggestion)
	}
}

func TestMapNetworkError_Variants(t *testing.T) {
	t.Run("timeout", func(t *testing.T) {
		err := &net.OpError{Op: "dial", Err: timeoutErr{}}
		got := MapNetworkError(err)
		if got.Reason != "NetworkTimeout" {
			t.Errorf("timeout reason = %q, want NetworkTimeout", got.Reason)
		}
		if got.ExitCode != ExitUnavailable {
			t.Errorf("network exit = %d, want %d", got.ExitCode, ExitUnavailable)
		}
	})
	t.Run("url error", func(t *testing.T) {
		err := &url.Error{Op: "Get", URL: "http://x", Err: errors.New("connection refused")}
		got := MapNetworkError(err)
		if got.Reason != "URLRequestError" {
			t.Errorf("url error reason = %q, want URLRequestError", got.Reason)
		}
	})
	t.Run("plain error", func(t *testing.T) {
		got := MapNetworkError(errors.New("dial tcp: connection refused"))
		if got.Reason != "NetworkError" {
			t.Errorf("plain error reason = %q, want NetworkError", got.Reason)
		}
	})
	t.Run("nil", func(t *testing.T) {
		if got := MapNetworkError(nil); got != nil {
			t.Errorf("nil input should return nil, got %+v", got)
		}
	})
}

func TestExitCodeFor_Table(t *testing.T) {
	cases := []struct {
		code      int
		isNetwork bool
		want      int
	}{
		{401, false, ExitNotLogged},
		{429, false, ExitRateLimit},
		{500, false, ExitUnavailable},
		{502, false, ExitUnavailable},
		{400, false, ExitBusiness},
		{0, true, ExitUnavailable},
		{401, true, ExitUnavailable},
	}
	for _, c := range cases {
		got := ExitCodeFor(c.code, c.isNetwork)
		if got != c.want {
			t.Errorf("ExitCodeFor(%d, %v) = %d, want %d", c.code, c.isNetwork, got, c.want)
		}
	}
}

func TestHandleCLIError_JSONOutput_AllFieldsPresent(t *testing.T) {
	if os.Getenv("TEST_JSON_ERROR") == "1" {
		oldFmt := outputFmt
		outputFmt = "json"
		defer func() { outputFmt = oldFmt }()
		handleCLIError(nil, &CLIError{
			StatusCode: 401, Reason: "Unauthorized",
			ServerMessage: "expired", FixSuggestion: "请重新登录", ExitCode: ExitNotLogged,
		})
		return
	}
	cmd := exec.Command(os.Args[0], "-test.run=TestHandleCLIError_JSONOutput_AllFieldsPresent")
	cmd.Env = append(os.Environ(), "TEST_JSON_ERROR=1")
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("subprocess should exit non-zero; output=%s", out)
	}
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	var jsonLine string
	for _, l := range lines {
		if strings.HasPrefix(strings.TrimSpace(l), "{") {
			jsonLine = l
			break
		}
	}
	if jsonLine == "" {
		t.Fatalf("no JSON line in output: %s", out)
	}
	var m map[string]any
	if err := json.Unmarshal([]byte(jsonLine), &m); err != nil {
		t.Fatalf("invalid JSON: %v (line=%q)", err, jsonLine)
	}
	for _, k := range []string{"status_code", "reason", "server_message", "fix_suggestion", "exit_code"} {
		if _, ok := m[k]; !ok {
			t.Errorf("JSON missing field %q: %s", k, jsonLine)
		}
	}
	if int(m["exit_code"].(float64)) != ExitNotLogged {
		t.Errorf("exit_code = %v, want %d", m["exit_code"], ExitNotLogged)
	}
}

func TestHandleCLIError_TextOutput_Format(t *testing.T) {
	if os.Getenv("TEST_TEXT_ERROR") == "1" {
		oldFmt := outputFmt
		outputFmt = "table"
		defer func() { outputFmt = oldFmt }()
		root := &cobra.Command{Use: "tick"}
		sub := &cobra.Command{Use: "task"}
		list := &cobra.Command{Use: "list"}
		root.AddCommand(sub)
		sub.AddCommand(list)
		handleCLIError(list, &CLIError{
			StatusCode: 404, Reason: "Not Found",
			ServerMessage: "task t_xxx not found", FixSuggestion: "检查 ID", ExitCode: ExitBusiness,
		})
		return
	}
	cmd := exec.Command(os.Args[0], "-test.run=TestHandleCLIError_TextOutput_Format")
	cmd.Env = append(os.Environ(), "TEST_TEXT_ERROR=1")
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("subprocess should exit non-zero; output=%s", out)
	}
	text := string(out)
	if !strings.Contains(text, "tick task list: 404 Not Found:") {
		t.Errorf("text format mismatch: %s", text)
	}
	if !strings.Contains(text, "检查 ID") {
		t.Errorf("fix suggestion missing: %s", text)
	}
}

func TestParseRetryAfter(t *testing.T) {
	cases := []struct {
		in   string
		want int
	}{
		{"", 0},
		{"30", 30},
		{"0", 0},
		{"-5", 0},
		{"abc", 0},
		{"  42  ", 42},
	}
	for _, c := range cases {
		if got := parseRetryAfter(c.in); got != c.want {
			t.Errorf("parseRetryAfter(%q) = %d, want %d", c.in, got, c.want)
		}
	}
}

type timeoutErr struct{}

func (timeoutErr) Error() string   { return "i/o timeout" }
func (timeoutErr) Timeout() bool   { return true }
func (timeoutErr) Temporary() bool { return true }

// TestE2E_HandleCLIError_ExitCodesAcrossStatuses drives handleCLIError
// through a real subprocess per status code so the os.Exit contract is
// observable from the parent (FR-040, FR-041, SC-011/12).
func TestE2E_HandleCLIError_ExitCodesAcrossStatuses(t *testing.T) {
	cases := []struct {
		code int
		want int
	}{
		{http.StatusUnauthorized, ExitNotLogged},
		{http.StatusNotFound, ExitBusiness},
		{http.StatusUnprocessableEntity, ExitBusiness},
		{http.StatusTooManyRequests, ExitRateLimit},
		{http.StatusInternalServerError, ExitUnavailable},
		{http.StatusServiceUnavailable, ExitUnavailable},
		{http.StatusBadGateway, ExitUnavailable},
	}
	for _, c := range cases {
		c := c
		t.Run(fmt.Sprint(c.code), func(t *testing.T) {
			cmd := exec.Command(os.Args[0], "-test.run=TestE2E_HandleCLIError_ExitCodesAcrossStatuses")
			cmd.Env = append(os.Environ(),
				"BE_CROSSTEST_STATUS=1",
				fmt.Sprintf("TEST_STATUS=%d", c.code),
				fmt.Sprintf("TEST_WANT_EXIT=%d", c.want),
			)
			err := cmd.Run()
			if err == nil {
				t.Fatalf("expected non-zero exit")
			}
			ee, ok := err.(*exec.ExitError)
			if !ok {
				t.Fatalf("not ExitError: %v", err)
			}
			if ee.ExitCode() != c.want {
				t.Errorf("status %d → exit %d, want %d", c.code, ee.ExitCode(), c.want)
			}
		})
	}
}

// subprocess driver for the e2e test above.
func init() {
	if os.Getenv("BE_CROSSTEST_STATUS") == "1" {
		code, _ := strconv.Atoi(os.Getenv("TEST_STATUS"))
		oldFmt := outputFmt
		outputFmt = "table"
		defer func() { outputFmt = oldFmt }()
		handleCLIError(nil, MapHTTPError(code, []byte("body"), "30"))
	}
}

// TestE2E_RealHTTPServer_HappyPath verifies that a 2xx response does not
// trigger handleCLIError (so the subprocess exits 0). This guards against
// false positives in the error pipeline.
func TestE2E_RealHTTPServer_HappyPath(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()

	SetBuildMetadata("test", "c1", "now", srv.URL, srv.URL, "prod", "")
	t.Setenv("TICK_API_KEY", "tk_test")

	if os.Getenv("BE_HAPPY_DRIVER") == "1" {
		oldFmt := outputFmt
		outputFmt = "json"
		defer func() { outputFmt = oldFmt }()
		// Synthetic cmd for ctx-based key resolution
		cmd := &cobra.Command{Use: "list"}
		cmd.SetContext(context.WithValue(context.Background(), apiKeyContextKey{}, "tk_test"))
		printResp(doGet(cmd, "/api/v1/tasks"))
		return
	}
	cmd := exec.Command(os.Args[0], "-test.run=TestE2E_RealHTTPServer_HappyPath")
	cmd.Env = append(os.Environ(), "BE_HAPPY_DRIVER=1")
	if err := cmd.Run(); err != nil {
		t.Errorf("happy path should exit 0, got %v", err)
	}
}
