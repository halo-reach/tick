package worker

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/tickplatform/tick/internal/auth"
	"github.com/tickplatform/tick/internal/domain"
	"github.com/tickplatform/tick/internal/hook"
)

type ExecResult struct {
	Status         domain.ExecStatus
	StatusCode     int
	DurationMs     int
	RequestHeaders string
	RequestBody    string
	ResponseBody   string
	ErrorMsg       string
}

type HTTPExecutor struct {
	client *http.Client
}

func NewHTTPExecutor() *HTTPExecutor {
	return &HTTPExecutor{
		client: &http.Client{},
	}
}

func (e *HTTPExecutor) Execute(ctx context.Context, task *domain.Task, target *domain.Target, triggerTime time.Time, secret string, executionID string, injections []domain.CredentialInjection, vc *hook.VariableContext) ExecResult {
	var cfg domain.HTTPTargetConfig
	if err := json.Unmarshal(target.Config, &cfg); err != nil {
		return ExecResult{Status: domain.ExecFailed, ErrorMsg: "invalid target config: " + err.Error()}
	}

	if vc != nil {
		if rendered, err := hook.Render(cfg.URL, vc); err == nil {
			cfg.URL = rendered
		}
		for k, v := range cfg.Headers {
			if rendered, err := hook.Render(v, vc); err == nil {
				cfg.Headers[k] = rendered
			}
		}
		if cfg.Body != nil {
			if rendered, err := hook.RenderJSON(cfg.Body, vc); err == nil {
				cfg.Body = rendered
			}
		}
	}

	timeout := time.Duration(task.TimeoutSecs) * time.Second
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	var bodyReader io.Reader
	var reqBody string
	contentType := "application/json"
	if cfg.Method == "POST" || cfg.Method == "PUT" || cfg.Method == "PATCH" {
		if cfg.ContentType == "form" && cfg.Body != nil {
			var kvs map[string]string
			if json.Unmarshal(cfg.Body, &kvs) == nil {
				form := url.Values{}
				for k, v := range kvs {
					form.Set(k, v)
				}
				encoded := form.Encode()
				reqBody = truncate(encoded, 4096)
				bodyReader = strings.NewReader(encoded)
				contentType = "application/x-www-form-urlencoded"
			}
		} else if cfg.Body != nil {
			body := cfg.Body
			var s string
			if json.Unmarshal(body, &s) == nil {
				body = []byte(s)
			}
			reqBody = truncate(string(body), 4096)
			bodyReader = bytes.NewReader(body)
		} else {
			payload := map[string]any{
				"task_id":      task.ID,
				"task_name":    task.Name,
				"trigger_time": triggerTime.Format(time.RFC3339),
				"attempt":      1,
			}
			b, _ := json.Marshal(payload)
			reqBody = truncate(string(b), 4096)
			bodyReader = bytes.NewReader(b)
		}
	}

	req, err := http.NewRequestWithContext(ctx, cfg.Method, cfg.URL, bodyReader)
	if err != nil {
		return ExecResult{Status: domain.ExecFailed, ErrorMsg: "create request: " + err.Error(), RequestBody: reqBody}
	}
	req.Header.Set("Content-Type", contentType)
	req.Header.Set("X-Tick-Task-ID", task.ID)
	ts := strconv.FormatInt(triggerTime.Unix(), 10)
	req.Header.Set("X-Tick-Timestamp", ts)
	if secret != "" {
		req.Header.Set("X-Tick-Signature", auth.Sign(task.ID, ts, secret))
	}
	if executionID != "" {
		req.Header.Set("X-Tick-Execution-ID", executionID)
	}

	for k, v := range cfg.Headers {
		req.Header.Set(k, v)
	}

	for _, inj := range injections {
		switch inj.Location {
		case "header":
			req.Header[inj.Key] = []string{inj.Value}
		case "query":
			q := req.URL.Query()
			q.Set(inj.Key, inj.Value)
			req.URL.RawQuery = q.Encode()
		case "cookie":
			req.AddCookie(&http.Cookie{Name: inj.Key, Value: inj.Value})
		}
	}

	reqHeaders := marshalHeaders(req.Header)

	start := time.Now()
	resp, err := e.client.Do(req)
	durationMs := int(time.Since(start).Milliseconds())

	if err != nil {
		status := domain.ExecFailed
		if ctx.Err() != nil {
			status = domain.ExecTimeout
		}
		return ExecResult{Status: status, DurationMs: durationMs, ErrorMsg: err.Error(), RequestHeaders: reqHeaders, RequestBody: reqBody}
	}
	defer resp.Body.Close()

	respBytes, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
	respBody := truncate(string(respBytes), 4096)

	result := ExecResult{
		StatusCode:     resp.StatusCode,
		DurationMs:     durationMs,
		RequestHeaders: reqHeaders,
		RequestBody:    reqBody,
		ResponseBody:   respBody,
	}
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		result.Status = domain.ExecSuccess
	} else {
		result.Status = domain.ExecFailed
		result.ErrorMsg = fmt.Sprintf("HTTP %d", resp.StatusCode)
	}
	return result
}

func truncate(s string, max int) string {
	if len(s) > max {
		return s[:max]
	}
	return s
}

func marshalHeaders(h http.Header) string {
	flat := make(map[string]string, len(h))
	for k, v := range h {
		if len(v) == 1 {
			flat[k] = v[0]
		} else {
			b, _ := json.Marshal(v)
			flat[k] = string(b)
		}
	}
	b, _ := json.Marshal(flat)
	return truncate(string(b), 4096)
}
