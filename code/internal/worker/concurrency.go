package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

type ConcurrencyGuard struct {
	rdb *redis.Client
}

func NewConcurrencyGuard(rdb *redis.Client) *ConcurrencyGuard {
	return &ConcurrencyGuard{rdb: rdb}
}

func inflightKey(taskID string) string {
	return fmt.Sprintf("tick:inflight:%s", taskID)
}

func queueKey(taskID string) string {
	return fmt.Sprintf("tick:queue:%s", taskID)
}

func (g *ConcurrencyGuard) Acquire(ctx context.Context, taskID string, maxConcurrency int, ttl time.Duration) (bool, error) {
	key := inflightKey(taskID)
	val, err := g.rdb.Incr(ctx, key).Result()
	if err != nil {
		return false, err
	}
	if int(val) > maxConcurrency {
		g.rdb.Decr(ctx, key)
		return false, nil
	}
	// Always refresh TTL so the key expires even if the process crashes
	g.rdb.Expire(ctx, key, ttl)
	return true, nil
}

func (g *ConcurrencyGuard) Release(ctx context.Context, taskID string) error {
	key := inflightKey(taskID)
	val, err := g.rdb.Decr(ctx, key).Result()
	if err != nil {
		return err
	}
	if val <= 0 {
		g.rdb.Del(ctx, key)
	}
	return nil
}

const maxQueueDepth = 5

func (g *ConcurrencyGuard) QueuePush(ctx context.Context, taskID string, payload []byte) (bool, error) {
	key := queueKey(taskID)
	length, err := g.rdb.LLen(ctx, key).Result()
	if err != nil {
		return false, err
	}
	if length >= maxQueueDepth {
		return false, nil
	}
	return true, g.rdb.LPush(ctx, key, payload).Err()
}

func (g *ConcurrencyGuard) QueuePop(ctx context.Context, taskID string) ([]byte, error) {
	key := queueKey(taskID)
	val, err := g.rdb.RPop(ctx, key).Bytes()
	if err == redis.Nil {
		return nil, nil
	}
	return val, err
}

type QueuedTrigger struct {
	TaskID      string    `json:"task_id"`
	TenantID    string    `json:"tenant_id"`
	TriggerTime time.Time `json:"trigger_time"`
	ExecutionID string    `json:"execution_id"`
}

func MarshalQueuedTrigger(t QueuedTrigger) []byte {
	b, _ := json.Marshal(t)
	return b
}

func UnmarshalQueuedTrigger(data []byte) (QueuedTrigger, error) {
	var t QueuedTrigger
	err := json.Unmarshal(data, &t)
	return t, err
}
