package hook

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/tidwall/gjson"
	"github.com/tickplatform/tick/internal/credential"
	"github.com/tickplatform/tick/internal/domain"
)

type Engine struct {
	resolver *credential.Resolver
	client   *http.Client
}

func NewEngine(resolver *credential.Resolver) *Engine {
	return &Engine{
		resolver: resolver,
		client:   &http.Client{Timeout: 30 * time.Second},
	}
}

func (e *Engine) ExecutePreHooks(ctx context.Context, hooks []domain.PreHook, vc *VariableContext, tenantID string) ([]domain.HookResultEntry, []domain.CredentialInjection, error) {
	var results []domain.HookResultEntry
	var injections []domain.CredentialInjection

	for i, h := range hooks {
		start := time.Now()
		entry := domain.HookResultEntry{Index: i, Type: h.Type, Status: "success"}

		switch h.Type {
		case domain.HookTypeCredential:
			resolved, err := e.resolver.Resolve(ctx, h.CredentialID, tenantID)
			if err != nil {
				entry.Status = "failed"
				entry.ErrorMsg = err.Error()
				entry.DurationMs = int(time.Since(start).Milliseconds())
				results = append(results, entry)
				return results, injections, fmt.Errorf("pre-hook[%d] credential resolve: %w", i, err)
			}
			if h.Inject != nil {
				value := resolved.Token
				if resolved.Secret != "" {
					value = resolved.Secret
				}
				if h.Inject.Prefix != "" {
					value = h.Inject.Prefix + value
				}
				if resolved.Type == domain.CredTypeCustomHeader {
					for k, v := range resolved.Headers {
						injections = append(injections, domain.CredentialInjection{Location: h.Inject.Location, Key: k, Value: v})
					}
				} else {
					injections = append(injections, domain.CredentialInjection{Location: h.Inject.Location, Key: h.Inject.Key, Value: value})
				}
			}

		case domain.HookTypeHTTP:
			if h.Request == nil {
				entry.Status = "failed"
				entry.ErrorMsg = "missing request config"
				entry.DurationMs = int(time.Since(start).Milliseconds())
				results = append(results, entry)
				return results, injections, fmt.Errorf("pre-hook[%d]: missing request config", i)
			}

			reqURL, err := Render(h.Request.URL, vc)
			if err != nil {
				entry.Status = "failed"
				entry.ErrorMsg = err.Error()
				entry.DurationMs = int(time.Since(start).Milliseconds())
				results = append(results, entry)
				return results, injections, fmt.Errorf("pre-hook[%d] render URL: %w", i, err)
			}

			var bodyReader io.Reader
			if h.Request.Body != nil {
				rendered, err := RenderJSON(h.Request.Body, vc)
				if err != nil {
					entry.Status = "failed"
					entry.ErrorMsg = err.Error()
					entry.DurationMs = int(time.Since(start).Milliseconds())
					results = append(results, entry)
					return results, injections, fmt.Errorf("pre-hook[%d] render body: %w", i, err)
				}
				bodyReader = strings.NewReader(string(rendered))
			}

			timeout := 30 * time.Second
			if h.TimeoutSecs > 0 {
				timeout = time.Duration(h.TimeoutSecs) * time.Second
			}
			hookCtx, cancel := context.WithTimeout(ctx, timeout)

			method := h.Request.Method
			if method == "" {
				method = "GET"
			}
			req, err := http.NewRequestWithContext(hookCtx, method, reqURL, bodyReader)
			if err != nil {
				cancel()
				entry.Status = "failed"
				entry.ErrorMsg = err.Error()
				entry.DurationMs = int(time.Since(start).Milliseconds())
				results = append(results, entry)
				return results, injections, fmt.Errorf("pre-hook[%d] create request: %w", i, err)
			}

			for k, v := range h.Request.Headers {
				rendered, _ := Render(v, vc)
				req.Header.Set(k, rendered)
			}

			resp, err := e.client.Do(req)
			cancel()
			if err != nil {
				entry.Status = "failed"
				entry.ErrorMsg = err.Error()
				entry.DurationMs = int(time.Since(start).Milliseconds())
				results = append(results, entry)
				return results, injections, fmt.Errorf("pre-hook[%d] HTTP request: %w", i, err)
			}

			respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 65536))
			resp.Body.Close()

			if resp.StatusCode >= 300 {
				entry.Status = "failed"
				entry.ErrorMsg = fmt.Sprintf("HTTP %d", resp.StatusCode)
				entry.DurationMs = int(time.Since(start).Milliseconds())
				results = append(results, entry)
				return results, injections, fmt.Errorf("pre-hook[%d] HTTP %d", i, resp.StatusCode)
			}

			if h.Extract != nil {
				val := gjson.GetBytes(respBody, h.Extract.Path).String()
				vc.Set(h.Extract.As, val)
				entry.Extracted = map[string]string{h.Extract.As: val}
			}
		}

		entry.DurationMs = int(time.Since(start).Milliseconds())
		results = append(results, entry)
	}

	return results, injections, nil
}

func (e *Engine) ExecutePostHooks(ctx context.Context, hooks []domain.PostHook, vc *VariableContext, execStatus string) []domain.HookResultEntry {
	var results []domain.HookResultEntry

	for i, h := range hooks {
		switch h.When {
		case domain.HookWhenSuccess:
			if execStatus != "success" {
				continue
			}
		case domain.HookWhenFailure:
			if execStatus != "failed" {
				continue
			}
		}

		start := time.Now()
		entry := domain.HookResultEntry{Index: i, Type: h.Type, When: h.When, Status: "success"}

		timeout := 30 * time.Second
		if h.TimeoutSecs > 0 {
			timeout = time.Duration(h.TimeoutSecs) * time.Second
		}

		switch h.Type {
		case domain.HookTypeHTTP:
			if h.Request == nil {
				entry.Status = "failed"
				entry.ErrorMsg = "missing request config"
				entry.DurationMs = int(time.Since(start).Milliseconds())
				results = append(results, entry)
				continue
			}

			reqURL, err := Render(h.Request.URL, vc)
			if err != nil {
				entry.Status = "failed"
				entry.ErrorMsg = err.Error()
				entry.DurationMs = int(time.Since(start).Milliseconds())
				results = append(results, entry)
				continue
			}

			var bodyReader io.Reader
			if h.Request.Body != nil {
				rendered, err := RenderJSON(h.Request.Body, vc)
				if err != nil {
					entry.Status = "failed"
					entry.ErrorMsg = err.Error()
					entry.DurationMs = int(time.Since(start).Milliseconds())
					results = append(results, entry)
					continue
				}
				bodyReader = strings.NewReader(string(rendered))
			}

			hookCtx, cancel := context.WithTimeout(ctx, timeout)
			method := h.Request.Method
			if method == "" {
				method = "POST"
			}
			req, err := http.NewRequestWithContext(hookCtx, method, reqURL, bodyReader)
			if err != nil {
				cancel()
				entry.Status = "failed"
				entry.ErrorMsg = err.Error()
				entry.DurationMs = int(time.Since(start).Milliseconds())
				results = append(results, entry)
				continue
			}
			for k, v := range h.Request.Headers {
				rendered, _ := Render(v, vc)
				req.Header.Set(k, rendered)
			}

			resp, err := e.client.Do(req)
			cancel()
			if err != nil {
				entry.Status = "failed"
				entry.ErrorMsg = err.Error()
				entry.DurationMs = int(time.Since(start).Milliseconds())
				results = append(results, entry)
				continue
			}
			respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 65536))
			resp.Body.Close()

			if resp.StatusCode >= 300 {
				entry.Status = "failed"
				entry.ErrorMsg = fmt.Sprintf("HTTP %d", resp.StatusCode)
			}

			if len(h.ResponseExtract) > 0 {
				extracted := make(map[string]string)
				for _, ext := range h.ResponseExtract {
					val := gjson.GetBytes(respBody, ext.Path).String()
					vc.Set(ext.As, val)
					extracted[ext.As] = val
				}
				entry.Extracted = extracted
			}

		case domain.HookTypeFeishu:
			if h.Request == nil {
				entry.Status = "failed"
				entry.ErrorMsg = "missing request config"
				entry.DurationMs = int(time.Since(start).Milliseconds())
				results = append(results, entry)
				continue
			}

			reqURL, err := Render(h.Request.URL, vc)
			if err != nil {
				entry.Status = "failed"
				entry.ErrorMsg = err.Error()
				entry.DurationMs = int(time.Since(start).Milliseconds())
				results = append(results, entry)
				continue
			}

			var body json.RawMessage
			if h.Request.Body != nil {
				rendered, err := RenderJSON(h.Request.Body, vc)
				if err != nil {
					entry.Status = "failed"
					entry.ErrorMsg = err.Error()
					entry.DurationMs = int(time.Since(start).Milliseconds())
					results = append(results, entry)
					continue
				}
				body = rendered
			}

			_, err = ExecuteFeishu(ctx, reqURL, body, timeout)
			if err != nil {
				entry.Status = "failed"
				entry.ErrorMsg = err.Error()
			}
		}

		entry.DurationMs = int(time.Since(start).Milliseconds())
		results = append(results, entry)
	}

	return results
}
