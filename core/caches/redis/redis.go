package redis

import (
	"fmt"
	"time"

	"github.com/garyburd/redigo/redis"
)

// Redis is a cache backend.
type Redis struct {
	pool   *redis.Pool
	prefix string
}

// New Redis backend.
func New(address string, db int, prefix string) (*Redis, error) {
	r := &Redis{
		prefix: prefix,
		pool: &redis.Pool{
			MaxIdle:     3,
			IdleTimeout: 240 * time.Second,
			Dial: func() (redis.Conn, error) {
				return redis.Dial("tcp", address,
					redis.DialConnectTimeout(10*time.Second),
					redis.DialDatabase(db),
					redis.DialKeepAlive(30*time.Second),
				)
			},
		},
	}

	return r, nil
}

// Get a key from Redis.
func (r *Redis) Get(key string) ([]byte, error) {
	conn := r.pool.Get()

	n, err := redis.Bytes(conn.Do("GET", r.prefix+key))
	if err != nil {
		return nil, fmt.Errorf("Redis SET error: %s (conn error: %s)", err, conn.Err())
	}
	return n, err
}

// Set a key in Redis.
func (r *Redis) Set(key string, v []byte, expires int) error {
	conn := r.pool.Get()

	_, err := conn.Do("SETEX", r.prefix+key, expires, v)
	if err != nil {
		return fmt.Errorf("Redis SET error: %s (conn error: %s)", err, conn.Err())
	}
	return nil
}

// Delete keys from Redis.
func (r *Redis) Delete(key ...string) {
	conn := r.pool.Get()

	for _, v := range key {
		conn.Send("DEL", v)
	}
	conn.Flush()
	conn.Receive()
}

// Keys returns a list of Redis keys matching a pattern.
func (r *Redis) Keys(pattern string) ([]string, error) {
	conn := r.pool.Get()

	n, err := redis.Strings(conn.Do("KEYS", pattern))
	return n, err
}

// Lock a mutex.
func (r *Redis) Lock(key, value string, expires int) bool {
	conn := r.pool.Get()

	n, err := redis.Int(conn.Do("SETNX", r.prefix+key, value))
	return err == nil && n == 1
}

// Unlock a mutex.
func (r *Redis) Unlock(key string) {
	r.Delete(key)
}
