package e2e

import (
	"fmt"
	"testing"
	"time"
)

// TestQuickstartValidation validates the quickstart.md flow end-to-end:
// register → create cron task via API → wait → check history → cleanup
func TestQuickstartValidation(t *testing.T) {
	if testing.Short() {
		t.Skip("e2e test")
	}

	// Step 1: Register tenant (quickstart step 2)
	c := newClient(t)
	if c.apiKey == "" {
		t.Fatal("registration failed: no api_key returned")
	}
	t.Log("✓ Tenant registered, got API key")

	// Step 2: Create a cron task (quickstart step 4 - every 5 seconds for test speed)
	task := c.post(t, "/api/v1/tasks", map[string]any{
		"name":          "quickstart-health-check",
		"schedule_type": "cron",
		"cron_expr":     "*/5 * * * * *",
		"url":           "https://httpbin.org/post",
	})
	taskID, ok := task["id"].(string)
	if !ok || taskID == "" {
		t.Fatalf("task creation failed: %+v", task)
	}
	t.Logf("✓ Task created: %s", taskID)

	// Step 3: List tasks
	tasks := c.get(t, "/api/v1/tasks")
	if tasks == nil {
		t.Fatal("task list returned nil")
	}
	t.Log("✓ Task list works")

	// Step 4: Wait for execution
	time.Sleep(12 * time.Second)

	// Step 5: Check execution history (quickstart step 4 - tick task history)
	history := c.get(t, fmt.Sprintf("/api/v1/tasks/%s/history", taskID))
	if history["data"] == nil {
		t.Fatal("no execution history after waiting")
	}
	t.Log("✓ Execution history has records")

	// Cleanup
	c.delete(t, "/api/v1/tasks/"+taskID)
	t.Log("✓ Quickstart flow validated end-to-end")
}
