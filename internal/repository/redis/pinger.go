package redis

import (
	"context"

	"github.com/redis/go-redis/v9"
)

// RedisPinger adapta *redis.Client a la interfaz handlers.Pinger.
type RedisPinger struct {
	client *redis.Client
}

func NewPinger(client *redis.Client) *RedisPinger {
	return &RedisPinger{client: client}
}

func (p *RedisPinger) Ping(ctx context.Context) error {
	return p.client.Ping(ctx).Err()
}
