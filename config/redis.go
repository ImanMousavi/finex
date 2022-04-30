package config

import (
	"context"
	"encoding/json"
	"os"
	"time"

	"github.com/go-redis/redis/v8"
)

var (
	Redis *CacheService
)

type CacheService struct {
	Ctx        context.Context
	Connection *redis.Client
}

func NewCacheService() error {
	c := redis.NewClient(&redis.Options{
		Addr:     os.Getenv("REDIS_HOST") + ":" + os.Getenv("REDIS_PORT"),
		Username: os.Getenv("REDIS_USERNAME"),
		Password: os.Getenv("REDIS_PASSWORD"),
		DB:       0,
	})
	ctx := context.Background()

	if err := c.Ping(ctx).Err(); err != nil {
		return err
	}

	Redis = &CacheService{
		Ctx:        ctx,
		Connection: c,
	}

	return nil
}

//GetKey get key
func (c *CacheService) GetKey(key string, src interface{}) error {
	val, err := c.Connection.Get(c.Ctx, key).Result()
	if err == redis.Nil || err != nil {
		return err
	}
	err = json.Unmarshal([]byte(val), src)
	if err != nil {
		return err
	}
	return nil
}

//SetKey set key
func (c *CacheService) SetKey(key string, value interface{}, expiration time.Duration) error {
	cacheEntry, err := json.Marshal(value)
	if err != nil {
		return err
	}
	err = c.Connection.Set(c.Ctx, key, cacheEntry, expiration).Err()
	if err != nil {
		return err
	}
	return nil
}
