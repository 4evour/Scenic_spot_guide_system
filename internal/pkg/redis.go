package pkg

import (
	"context"
	"log/slog"
	"sync"
	"sync/atomic"

	"github.com/redis/go-redis/v9"
	"github.com/scenic-guide/internal/config"
)

var (
	redisClient atomic.Pointer[redis.Client]
	redisOnce   sync.Once
)

// InitRedis connects to Redis using the given config.
// If cfg is nil or Addr is empty, Redis is not initialised and GetRedis returns nil.
func InitRedis(cfg *config.RedisConfig) error {
	if cfg == nil || cfg.Addr == "" {
		slog.Info("Redis 未配置，将使用内存限流器")
		return nil
	}

	var initErr error
	redisOnce.Do(func() {
		client := redis.NewClient(&redis.Options{
			Addr:     cfg.Addr,
			Password: cfg.Password,
			DB:       cfg.DB,
		})

		ctx := context.Background()
		if err := client.Ping(ctx).Err(); err != nil {
			initErr = err
			return
		}

		redisClient.Store(client)
		slog.Info("Redis 连接成功", "addr", cfg.Addr)
	})

	return initErr
}

// GetRedis returns the package-level Redis client.
// Returns nil if Redis was not configured or failed to connect.
func GetRedis() *redis.Client {
	return redisClient.Load()
}
