package null

// Null is a cache that doesn't do anything.
type Null struct{}

// New Null cache backend.
func New() *Null {
	return &Null{}
}

// Get a key from Null.
func (r *Null) Get(key string) ([]byte, error) {
	return []byte{}, nil
}

// Set a key in Null.
func (r *Null) Set(key string, v []byte, expires int) error {
	return nil
}

// Delete keys from Null.
func (r *Null) Delete(key ...string) {}

// Keys returns a list of Null keys matching a pattern.
func (r *Null) Keys(pattern string) ([]string, error) {
	return []string{}, nil
}

// Lock a mutex.
func (r *Null) Lock(key string, value string, expires int) bool {
	return true
}

// Unlock a mutex.
func (r *Null) Unlock(key string) {}
