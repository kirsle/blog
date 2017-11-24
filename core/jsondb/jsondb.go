// Package jsondb implements a flat file JSON database engine.
package jsondb

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

// DB is the database manager.
type DB struct {
	Root string // The root directory of the database

	// Use Redis to cache filesystem reads of the database.
	EnableRedis bool
	RedisURL    string
}

// Error codes returned.
var (
	ErrNotFound = errors.New("document not found")
)

// New initializes the JSON database.
func New(root string) *DB {
	return &DB{
		Root: root,
	}
}

// Get a document by path and load it into the object `v`.
func (db *DB) Get(document string, v interface{}) error {
	log.Debug("[JsonDB] GET %s", document)
	if !db.Exists(document) {
		return ErrNotFound
	}

	// Get the file path and stats.
	path := db.toPath(document)
	_, err := os.Stat(path) // TODO: mtime for caching
	if err != nil {
		return err
	}

	// Read the JSON.
	err = db.readJSON(path, &v)
	if err != nil {
		return err
	}

	return nil
}

// Commit writes a JSON object to the database.
func (db *DB) Commit(document string, v interface{}) error {
	log.Debug("[JsonDB] COMMIT %s", document)
	path := db.toPath(document)

	// Ensure the directory tree is ready.
	db.makePath(path)

	// Write the document.
	err := db.writeJSON(path, v)
	if err != nil {
		return err
	}

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
	directory := filepath.Join(parts...)

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
	return filepath.Join(db.Root, document+".json")
}
