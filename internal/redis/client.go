package redis

import (
	"context"
	"fmt"
	"log"
	"time"

	"emergency-msystem-backend/config"

	"github.com/redis/go-redis/v9"
)

var Client *redis.Client

// Init 初始化 Redis 客户端
func Init(cfg config.RedisConfig) (*redis.Client, error) {
	rdb := redis.NewClient(&redis.Options{
		Addr:         cfg.Addr,
		Password:     cfg.Password,
		DB:           cfg.DB,
		PoolSize:     cfg.PoolSize,
		MinIdleConns: cfg.MinIdleConns,
		DialTimeout:  cfg.DialTimeout,
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if _, err := rdb.Ping(ctx).Result(); err != nil {
		return nil, fmt.Errorf("redis ping failed: %w", err)
	}

	Client = rdb
	log.Println("[Redis] Connection established")
	return rdb, nil
}

// Close 关闭 Redis 连接
func Close() {
	if Client != nil {
		Client.Close()
		log.Println("[Redis] Connection closed")
	}
}

// Key prefixes
const (
	KeySessionPrefix  = "session:"
	KeyMatchQueue     = "match:queue"
	KeyGPSTrackPrefix = "gps:track:"
	KeyRateLimit      = "ratelimit:"
	KeyTimeoutPrefix  = "timeout:"
)
