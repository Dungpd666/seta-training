package initialize

import (
	"context"
	"fmt"

	"github.com/redis/go-redis/v9"
)

func Redis(url string) (*redis.Client, error) {
	opt, err := redis.ParseURL(url)
	if err != nil {
		return nil, fmt.Errorf("parse REDIS_URL: %w", err)
	}
	rdb := redis.NewClient(opt)
	if err := rdb.Ping(context.Background()).Err(); err != nil {
		return nil, fmt.Errorf("ping: %w", err)
	}
	return rdb, nil
}
