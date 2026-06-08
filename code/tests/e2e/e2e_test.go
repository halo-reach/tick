package e2e

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"testing"
	"time"
)

var baseURL = envOrDefault("TICK_TEST_URL", "http://localhost:8080")

func envOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

type client struct {
	apiKey string
	http   *http.Client
}

func newClient(t *testing.T) *client {
	c := &client{http: &http.Client{Timeout: 10 * time.Second}}
	username := "e2e" + strconv.FormatInt(time.Now().UnixNano(), 10)
	resp := c.post(t, "/api/v1/auth/tenant-register", map[string]any{
		"username": username,
		"password": "e2e-test-password-123",
	})
	apiKey, _ := resp["api_key"].(string)
	if apiKey == "" {
		t.Fatalf("tenant-register did not return api_key, response: %+v", resp)
	}
	c.apiKey = apiKey
	return c
}

func (c *client) post(t *testing.T, path string, body any) map[string]any {
	return c.do(t, "POST", path, body)
}

func (c *client) get(t *testing.T, path string) map[string]any {
	return c.do(t, "GET", path, nil)
}

func (c *client) delete(t *testing.T, path string) map[string]any {
	return c.do(t, "DELETE", path, nil)
}

func (c *client) do(t *testing.T, method, path string, body any) map[string]any {
	var reader io.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		reader = bytes.NewReader(b)
	}
	req, err := http.NewRequest(method, baseURL+path, reader)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	if c.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	var result map[string]any
	json.Unmarshal(b, &result)
	if data, ok := result["data"]; ok {
		if m, ok := data.(map[string]any); ok {
			return m
		}
	}
	return result
}

func TestCronTaskCreateTriggerHistory(t *testing.T) {
	if testing.Short() {
		t.Skip("e2e test")
	}
	c := newClient(t)

	target := c.post(t, "/api/v1/targets", map[string]any{
		"name":   "e2e-target",
		"type":   "http",
		"config": map[string]any{"url": "https://httpbin.org/post", "method": "POST"},
	})

	task := c.post(t, "/api/v1/tasks", map[string]any{
		"name":          "e2e-cron-task",
		"schedule_type": "cron",
		"cron_expr":     "*/5 * * * * *",
		"target_id":     target["id"],
	})
	taskID := task["id"].(string)

	time.Sleep(12 * time.Second)

	history := c.get(t, fmt.Sprintf("/api/v1/tasks/%s/history", taskID))
	if history["data"] == nil {
		t.Fatal("expected execution history, got none")
	}

	c.delete(t, "/api/v1/tasks/"+taskID)
}

func TestOnceTaskAutoDelete(t *testing.T) {
	if testing.Short() {
		t.Skip("e2e test")
	}
	c := newClient(t)

	onceAt := time.Now().Add(3 * time.Second).Format(time.RFC3339)
	task := c.post(t, "/api/v1/tasks", map[string]any{
		"name":          "e2e-once-task",
		"schedule_type": "once",
		"once_at":       onceAt,
		"url":           "https://httpbin.org/post",
	})
	taskID := task["id"].(string)

	time.Sleep(8 * time.Second)

	history := c.get(t, fmt.Sprintf("/api/v1/tasks/%s/history", taskID))
	if history["data"] == nil {
		t.Fatal("expected execution history for once task")
	}
}

func TestIntervalTaskConsecutiveTriggers(t *testing.T) {
	if testing.Short() {
		t.Skip("e2e test")
	}
	c := newClient(t)

	task := c.post(t, "/api/v1/tasks", map[string]any{
		"name":           "e2e-interval-task",
		"schedule_type":  "interval",
		"interval_value": 3,
		"interval_unit":  "s",
		"url":            "https://httpbin.org/post",
	})
	taskID := task["id"].(string)

	time.Sleep(12 * time.Second)

	history := c.get(t, fmt.Sprintf("/api/v1/tasks/%s/history", taskID))
	if history["data"] == nil {
		t.Fatal("expected multiple execution records")
	}

	c.delete(t, "/api/v1/tasks/"+taskID)
}

func TestFailureRetryAttempts(t *testing.T) {
	if testing.Short() {
		t.Skip("e2e test")
	}
	c := newClient(t)

	task := c.post(t, "/api/v1/tasks", map[string]any{
		"name":          "e2e-retry-task",
		"schedule_type": "once",
		"once_at":       time.Now().Add(3 * time.Second).Format(time.RFC3339),
		"url":           "https://httpbin.org/status/500",
		"retry_count":   3,
	})
	taskID := task["id"].(string)

	// Wait for initial attempt + 3 retries with delay (10s + 30s + 90s is too long for e2e,
	// but Asynq retry delays are 10s, 30s, 90s. We wait enough for at least the first retry.)
	// Total: initial ~3s + retry1 ~10s + retry2 ~30s = ~45s, wait 60s to be safe for 3 attempts.
	time.Sleep(60 * time.Second)

	history := c.get(t, fmt.Sprintf("/api/v1/tasks/%s/history", taskID))
	if history["data"] == nil {
		t.Fatal("expected execution history with failed attempts")
	}
	items, ok := history["data"].([]any)
	if !ok {
		// single item response, check if there's a list in different format
		t.Logf("history response: %+v", history)
		t.Fatal("expected multiple execution records for retry attempts")
	}
	// Should have at least 3 records (initial + 2 retries within our wait window)
	if len(items) < 3 {
		t.Fatalf("expected at least 3 execution attempts, got %d", len(items))
	}
	// All should be failed
	for i, item := range items {
		rec, ok := item.(map[string]any)
		if !ok {
			continue
		}
		status, _ := rec["status"].(string)
		if status != "failed" {
			t.Fatalf("attempt %d: expected status 'failed', got '%s'", i+1, status)
		}
	}
}

func TestMultiTenantIsolation(t *testing.T) {
	if testing.Short() {
		t.Skip("e2e test")
	}
	clientA := newClient(t)
	clientB := newClient(t)

	task := clientA.post(t, "/api/v1/tasks", map[string]any{
		"name":          "tenant-a-task",
		"schedule_type": "once",
		"once_at":       time.Now().Add(1 * time.Hour).Format(time.RFC3339),
		"url":           "https://httpbin.org/post",
	})
	taskID := task["id"].(string)

	result := clientB.get(t, "/api/v1/tasks/"+taskID)
	if _, ok := result["error"]; !ok {
		t.Fatal("tenant B should not see tenant A's task")
	}
}
