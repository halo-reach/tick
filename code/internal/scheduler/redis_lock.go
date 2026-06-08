package scheduler

import (
	"context"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

const lockKey = "tick:scheduler:leader"

var renewScript = redis.NewScript(`
if redis.call("GET", KEYS[1]) == ARGV[1] then
    return redis.call("EXPIRE", KEYS[1], ARGV[2])
else
    return 0
end
`)

var releaseScript = redis.NewScript(`
if redis.call("GET", KEYS[1]) == ARGV[1] then
    return redis.call("DEL", KEYS[1])
else
    return 0
end
`)

type RedisLock struct {
	rdb   *redis.Client
	value string
	ttl   time.Duration
}

func NewRedisLock(rdb *redis.Client, ttl time.Duration) *RedisLock {
	return &RedisLock{
		rdb:   rdb,
		value: uuid.New().String(),
		ttl:   ttl,
	}
}

func (l *RedisLock) Acquire(ctx context.Context) (bool, error) {
	ok, err := l.rdb.SetNX(ctx, lockKey, l.value, l.ttl).Result()
	return ok, err
}

func (l *RedisLock) Renew(ctx context.Context) (bool, error) {
	result, err := renewScript.Run(ctx, l.rdb, []string{lockKey}, l.value, int(l.ttl.Seconds())).Int()
	if err != nil {
		return false, err
	}
	return result == 1, nil
}

func (l *RedisLock) Release(ctx context.Context) error {
	_, err := releaseScript.Run(ctx, l.rdb, []string{lockKey}, l.value).Result()
	return err
}

func RunRenewal(ctx context.Context, cancel context.CancelFunc, lock *RedisLock) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			ok, err := lock.Renew(ctx)
			if err != nil || !ok {
				slog.Error("scheduler lock renewal failed, stopping scheduler", "error", err)
				cancel()
				return
			}
		}
	}
}
