package redismq

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/lishimeng/x/redis"
)

var ErrRedisUnavailable = errors.New("redis client unavailable")

// Client Redis List 队列（LPUSH / BRPOP）。
type Client struct {
	rdb    *redis.Client
	prefix string
}

// New 使用已有 Redis 客户端；prefix 为空时默认 "mq"。
func New(rdb *redis.Client, prefix string) *Client {
	if prefix == "" {
		prefix = "mq"
	}
	return &Client{rdb: rdb, prefix: prefix}
}

func (c *Client) key(queue string) string {
	return fmt.Sprintf("%s:%s", c.prefix, queue)
}

// Push 入队（LPUSH）。
func (c *Client) Push(ctx context.Context, queue string, payload []byte) error {
	if c == nil || c.rdb == nil {
		return ErrRedisUnavailable
	}
	return c.rdb.LPush(ctx, c.key(queue), payload).Err()
}

// Pop 出队（BRPOP）；timeout 为 0 表示永久阻塞。
func (c *Client) Pop(ctx context.Context, queue string, timeout time.Duration) ([]byte, error) {
	if c == nil || c.rdb == nil {
		return nil, ErrRedisUnavailable
	}
	res, err := c.rdb.BRPop(ctx, timeout, c.key(queue)).Result()
	if err != nil {
		return nil, err
	}
	if len(res) < 2 {
		return nil, errors.New("redis brpop: empty result")
	}
	return []byte(res[1]), nil
}
