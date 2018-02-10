package jsondb

import (
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"time"
)

var errCacheDisabled = errors.New("cache disabled")

// SetCache sets a cache key.
func (db *DB) SetCache(key, value string, expires int) error {
	if db.Cache == nil {
		return errCacheDisabled
	}
	return db.Cache.Set(key, []byte(value), expires)
}

// SetJSONCache caches a JSON object.
func (db *DB) SetJSONCache(key string, v interface{}, expires int) error {
	if db.Cache == nil {
		return errCacheDisabled
	}
	bytes, err := json.Marshal(v)
	if err != nil {
		return err
	}
	return db.SetCache(key, string(bytes), expires)
}

// GetCache gets a cache key.
func (db *DB) GetCache(key string) (string, error) {
	if db.Cache == nil {
		return "", errCacheDisabled
	}
	v, err := db.Cache.Get(key)
	return string(v), err
}

// DeleteCache deletes a cache key.
func (db *DB) DeleteCache(key string) error {
	if db.Cache == nil {
		return errCacheDisabled
	}
	db.Cache.Delete(key)
	return nil
}

// LockCache implements 'file locking' in your cache.
func (db *DB) LockCache(key string) bool {
	if db.Cache == nil {
		return true
	}

	log.Info("LockCache(%s)", key)

	var (
		// In seconds
		timeout = 5 * time.Second
		expire  = 20
	)

	identifier := fmt.Sprintf("%d", rand.Uint64())
	log.Info("id: %s", identifier)

	end := time.Now().Add(timeout)
	for time.Now().Before(end) {
		if ok := db.Cache.Lock("lock:"+key, identifier, expire); ok {
			log.Info("JsonDB: Acquired lock for %s", key)
			return true
		}
		time.Sleep(1 * time.Millisecond)
	}
	log.Error("JsonDB: lock timeout for %s", key)
	return false
}

// UnlockCache releases the lock on a cache key.
func (db *DB) UnlockCache(key string) {
	if db.Cache == nil {
		return
	}
	db.Cache.Unlock("lock:" + key)
}
