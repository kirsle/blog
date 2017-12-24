// Package jsondb implements a flat file JSON database engine.
package jsondb

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/kirsle/blog/core/caches"
)

var (
	// CacheTimeout is how long the Redis cache keys live for in seconds, default 2 hours.
	CacheTimeout = 60 * 60 * 2
	CacheLock    sync.RWMutex
)

// DB is the database manager.
type DB struct {
	Root  string        // The root directory of the database
	Cache caches.Cacher // A cacher for the JSON documents, i.e. Redis
}

// Error codes returned.
var (
	ErrNotFound = errors.New("document not found")
)

// New initializes the JSON database.
func New(root string) *DB {
	log.Info("Initialized JsonDB at root: %s", root)
	return &DB{
		Root: root,
	}
}

// WithCache configures a memory cacher for the JSON documents.
func (db *DB) WithCache(cache caches.Cacher) *DB {
	db.Cache = cache
	return db
}

// Get a document by path and load it into the object `v`.
func (db *DB) Get(document string, v interface{}) error {
	log.Debug("[JsonDB] GET %s", document)
	if !db.Exists(document) {
		return ErrNotFound
	}

	// Get the file path and stats.
	path := db.toPath(document)
	stat, err := os.Stat(path)
	if err != nil {
		return err
	}

	// Do we have it cached?
	data, err := db.GetCache(document)
	if err == nil {
		// Check if the cache is fresh.
		cachedTime, err2 := db.GetCache(document + "_mtime")
		if err2 == nil {
			modTime := stat.ModTime()
			mtime, _ := time.Parse(time.RFC3339Nano, cachedTime)
			if modTime.After(mtime) && !modTime.Equal(mtime) {
				log.Debug("[JsonDB] %s: On-disk file is newer than cache", document)
				db.DeleteCache(document)
				db.DeleteCache(document + "_mtime")
			} else {
				log.Debug("[JsonDB] %s: Returning cached copy", document)
				return json.Unmarshal([]byte(data), v)
			}
		}
	}

	// Get a lock for reading.
	CacheLock.RLock()

	// Read the JSON.
	err = db.readJSON(path, &v)
	if err != nil {
		CacheLock.RUnlock()
		return err
	}

	// Unlock & cache it.
	db.SetJSONCache(document, v, CacheTimeout)
	db.SetCache(document+"_mtime", stat.ModTime().Format(time.RFC3339Nano), CacheTimeout)
	CacheLock.RUnlock()

	return nil
}

// Commit writes a JSON object to the database.
func (db *DB) Commit(document string, v interface{}) error {
	log.Debug("[JsonDB] COMMIT %s", document)
	path := db.toPath(document)

	// Get a write lock for the cache.
	CacheLock.Lock()

	// Ensure the directory tree is ready.
	err := db.makePath(path)
	if err != nil {
		CacheLock.Unlock()
		return err
	}

	// Write the document.
	err = db.writeJSON(path, v)
	if err != nil {
		CacheLock.Unlock()
		return fmt.Errorf("failed to write JSON to path %s: %s", path, err.Error())
	}

	// Unlock & cache it.
	db.SetJSONCache(document, v, CacheTimeout)
	db.SetCache(document+"_mtime", time.Now().Format(time.RFC3339Nano), CacheTimeout)
	CacheLock.Unlock()

	return nil
}

// Delete removes a JSON document from the database.
func (db *DB) Delete(document string) error {
	log.Debug("[JsonDB] DELETE %s", document)
	path := db.toPath(document)

	if _, err := os.Stat(path); os.IsNotExist(err) {
		log.Warn("Delete document %s: did not exist")
		return nil
	}

	db.DeleteCache(document)
	return os.Remove(path)
}

// Exists tells you whether a document exists in the database.
func (db *DB) Exists(document string) bool {
	if _, err := os.Stat(db.toPath(document)); os.IsNotExist(err) {
		return false
	}
	return true
}

// List all the documents at the path given.
func (db *DB) List(path string) ([]string, error) {
	return db.list(path, false)
}

// ListAll recursively lists the documents at the path prefix given.
func (db *DB) ListAll(path string) ([]string, error) {
	return db.list(path, true)
}

// makePath ensures all the directory components in a document path exist.
// path: the filesystem path like from toPath().
func (db *DB) makePath(path string) error {
	parts := strings.Split(path, string(filepath.Separator))
	parts = parts[:len(parts)-1] // pop off the filename
	var directory string
	if path[0] == '/' {
		directory = "/" + filepath.Join(parts...)
	} else {
		directory = filepath.Join(parts...)
	}

	if _, err := os.Stat(directory); err != nil {
		log.Debug("[JsonDB] Create directory: %s", directory)
		err = os.MkdirAll(directory, 0755)
		return err
	}

	return nil
}

// list returns the documents under a path with optional recursion.
func (db *DB) list(path string, recursive bool) ([]string, error) {
	root := filepath.Join(db.Root, path)
	var docs []string

	files, err := ioutil.ReadDir(root)
	if err != nil {
		return docs, err
	}

	for _, file := range files {
		filePath := filepath.Join(root, file.Name())
		dbPath := filepath.Join(path, file.Name())
		if file.IsDir() && recursive {
			subfiles, err := db.list(dbPath, recursive)
			if err != nil {
				return docs, err
			}
			docs = append(docs, subfiles...)
			continue
		}

		if strings.HasSuffix(filePath, ".json") {
			name := strings.TrimSuffix(dbPath, ".json")
			docs = append(docs, name)
		}
	}

	return docs, nil
}

// readJSON reads a JSON file from disk.
func (db *DB) readJSON(path string, v interface{}) error {
	fh, err := os.Open(path)
	if err != nil {
		return err
	}
	defer fh.Close()

	decoder := json.NewDecoder(fh)
	err = decoder.Decode(&v)
	return err
}

// writeJSON writes a JSON document to disk.
func (db *DB) writeJSON(path string, v interface{}) error {
	fh, err := os.Create(path)
	if err != nil {
		return err
	}
	defer fh.Close()

	encoder := json.NewEncoder(fh)
	encoder.SetIndent("", "\t")
	encoder.Encode(v)

	return nil
}

// toPath translates a document name into a filesystem path.
func (db *DB) toPath(document string) string {
	path, err := filepath.Abs(filepath.Join(db.Root, document+".json"))
	if err != nil {
		log.Error("[JsonDB] toPath error: %s", err)
	}
	return path
}
