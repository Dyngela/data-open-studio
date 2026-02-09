package pkg

import (
	"api"
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisSet stores a value in Redis with a TTL. The value is JSON-serialized.
func RedisSet(key string, value any, ttl time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	data, err := json.Marshal(value)
	if err != nil {
		return err
	}

	return api.Redis.Set(ctx, key, data, ttl).Err()
}

// RedisGet retrieves a value from Redis and JSON-deserializes it into dest.
// Returns redis.Nil if the key does not exist.
func RedisGet(key string, dest any) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	data, err := api.Redis.Get(ctx, key).Bytes()
	if err != nil {
		return err
	}

	return json.Unmarshal(data, dest)
}

// RedisDelete removes a key from Redis.
func RedisDelete(key string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	return api.Redis.Del(ctx, key).Err()
}

// RedisExists checks whether a key exists in Redis.
func RedisExists(key string) (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	n, err := api.Redis.Exists(ctx, key).Result()
	if err != nil {
		return false, err
	}

	return n > 0, nil
}

// IsRedisNil returns true if the error is a redis key-not-found error.
func IsRedisNil(err error) bool {
	return errors.Is(err, redis.Nil)
}
