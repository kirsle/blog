package caches

// Cacher is an interface for a key/value cacher.
type Cacher interface {
	Get(key string) ([]byte, error)
	Set(key string, v []byte, expires int) error
	Delete(key ...string)
	Keys(pattern string) ([]string, error)
	Lock(key, value string, expires int) bool
	Unlock(key string)
}
