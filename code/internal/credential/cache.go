package credential

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

type TokenCache struct {
	rdb *redis.Client
}

func NewTokenCache(rdb *redis.Client) *TokenCache {
	return &TokenCache{rdb: rdb}
}

func (c *TokenCache) Get(ctx context.Context, credentialID string) (string, error) {
	return c.rdb.Get(ctx, fmt.Sprintf("cred:token:%s", credentialID)).Result()
}

func (c *TokenCache) Set(ctx context.Context, credentialID, token string, ttl time.Duration) error {
	return c.rdb.Set(ctx, fmt.Sprintf("cred:token:%s", credentialID), token, ttl).Err()
}

func (c *TokenCache) Delete(ctx context.Context, credentialID string) error {
	return c.rdb.Del(ctx, fmt.Sprintf("cred:token:%s", credentialID)).Err()
}
