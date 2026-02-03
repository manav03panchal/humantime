package storage

import (
	"encoding/json"
	"errors"

	badger "github.com/dgraph-io/badger/v4"
	"github.com/manav03panchal/humantime/internal/model"
)

var (
	// ErrKeyNotFound is returned when a key is not found in the database.
	ErrKeyNotFound = errors.New("key not found")
)

// IsErrKeyNotFound returns true if the error is a key not found error.
func IsErrKeyNotFound(err error) bool {
	return errors.Is(err, ErrKeyNotFound) || errors.Is(err, badger.ErrKeyNotFound)
}

// Get retrieves a value by key and unmarshals it into v.
func (d *DB) Get(key string, v model.Model) error {
	return d.db.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte(key))
		if err != nil {
			if errors.Is(err, badger.ErrKeyNotFound) {
				return ErrKeyNotFound
			}
			return err
		}

		return item.Value(func(val []byte) error {
			if err := json.Unmarshal(val, v); err != nil {
				return err
			}
			v.SetKey(key)
			return nil
		})
	})
}

// GetBytes retrieves raw bytes by key.
func (d *DB) GetBytes(key string) ([]byte, error) {
	var result []byte
	err := d.db.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte(key))
		if err != nil {
			if errors.Is(err, badger.ErrKeyNotFound) {
				return ErrKeyNotFound
			}
			return err
		}

		return item.Value(func(val []byte) error {
			result = make([]byte, len(val))
			copy(result, val)
			return nil
		})
	})
	return result, err
}

// Set stores a model in the database.
func (d *DB) Set(v model.Model) error {
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}

	return d.db.Update(func(txn *badger.Txn) error {
		return txn.Set([]byte(v.GetKey()), data)
	})
}

// SetBytes stores raw bytes with the given key.
func (d *DB) SetBytes(key string, data []byte) error {
	return d.db.Update(func(txn *badger.Txn) error {
		return txn.Set([]byte(key), data)
	})
}

// GetRaw retrieves a value by key and unmarshals it into v (non-Model interface).
func (d *DB) GetRaw(key string, v interface{}) error {
	return d.db.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte(key))
		if err != nil {
			if errors.Is(err, badger.ErrKeyNotFound) {
				return ErrKeyNotFound
			}
			return err
		}

		return item.Value(func(val []byte) error {
			return json.Unmarshal(val, v)
		})
	})
}

// SetRaw stores any JSON-serializable value.
func (d *DB) SetRaw(key string, v interface{}) error {
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}

	return d.db.Update(func(txn *badger.Txn) error {
		return txn.Set([]byte(key), data)
	})
}

// Delete removes a key from the database.
func (d *DB) Delete(key string) error {
	return d.db.Update(func(txn *badger.Txn) error {
		return txn.Delete([]byte(key))
	})
}

// Exists checks if a key exists in the database.
func (d *DB) Exists(key string) (bool, error) {
	var exists bool
	err := d.db.View(func(txn *badger.Txn) error {
		_, err := txn.Get([]byte(key))
		if err != nil {
			if errors.Is(err, badger.ErrKeyNotFound) {
				exists = false
				return nil
			}
			return err
		}
		exists = true
		return nil
	})
	return exists, err
}

// GetOrCreate atomically retrieves an existing value or creates a new one.
// The createFunc is called only if the key doesn't exist, and should return
// the model to be created. Returns (model, created, error).
func (d *DB) GetOrCreate(key string, existing model.Model, createFunc func() model.Model) (model.Model, bool, error) {
	var result model.Model
	var created bool

	err := d.db.Update(func(txn *badger.Txn) error {
		// Try to get existing
		item, err := txn.Get([]byte(key))
		if err == nil {
			// Key exists, unmarshal it
			return item.Value(func(val []byte) error {
				if err := json.Unmarshal(val, existing); err != nil {
					return err
				}
				existing.SetKey(key)
				result = existing
				created = false
				return nil
			})
		}

		if !errors.Is(err, badger.ErrKeyNotFound) {
			return err
		}

		// Key doesn't exist, create new
		newModel := createFunc()
		newModel.SetKey(key)

		data, err := json.Marshal(newModel)
		if err != nil {
			return err
		}

		if err := txn.Set([]byte(key), data); err != nil {
			return err
		}

		result = newModel
		created = true
		return nil
	})

	return result, created, err
}

// ListByPrefix retrieves all keys with the given prefix.
func (d *DB) ListByPrefix(prefix string) ([]string, error) {
	var keys []string
	err := d.db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.PrefetchValues = false
		it := txn.NewIterator(opts)
		defer it.Close()

		prefixBytes := []byte(prefix)
		for it.Seek(prefixBytes); it.ValidForPrefix(prefixBytes); it.Next() {
			item := it.Item()
			key := make([]byte, len(item.Key()))
			copy(key, item.Key())
			keys = append(keys, string(key))
		}
		return nil
	})
	return keys, err
}

// GetAllByPrefix retrieves all values with the given prefix.
func GetAllByPrefix[T model.Model](d *DB, prefix string, newFunc func() T) ([]T, error) {
	var results []T
	err := d.db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.PrefetchSize = 100
		it := txn.NewIterator(opts)
		defer it.Close()

		prefixBytes := []byte(prefix)
		for it.Seek(prefixBytes); it.ValidForPrefix(prefixBytes); it.Next() {
			item := it.Item()
			err := item.Value(func(val []byte) error {
				v := newFunc()
				if err := json.Unmarshal(val, v); err != nil {
					return err
				}
				v.SetKey(string(item.Key()))
				results = append(results, v)
				return nil
			})
			if err != nil {
				return err
			}
		}
		return nil
	})
	return results, err
}

// GetFilteredByPrefix retrieves values matching a filter predicate.
// This is more efficient than GetAllByPrefix when you need to filter results,
// as it avoids loading all records into memory before filtering.
// The filter function receives each item and returns true to include it.
// If limit > 0, iteration stops after collecting that many matching items.
func GetFilteredByPrefix[T model.Model](d *DB, prefix string, newFunc func() T, filter func(T) bool, limit int) ([]T, error) {
	var results []T
	err := d.db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.PrefetchSize = 100
		it := txn.NewIterator(opts)
		defer it.Close()

		prefixBytes := []byte(prefix)
		for it.Seek(prefixBytes); it.ValidForPrefix(prefixBytes); it.Next() {
			item := it.Item()
			err := item.Value(func(val []byte) error {
				v := newFunc()
				if err := json.Unmarshal(val, v); err != nil {
					return err
				}
				v.SetKey(string(item.Key()))
				if filter(v) {
					results = append(results, v)
				}
				return nil
			})
			if err != nil {
				return err
			}
			// Check if we've hit our limit
			if limit > 0 && len(results) >= limit {
				break
			}
		}
		return nil
	})
	return results, err
}

// CountByPrefix counts items matching a filter predicate without loading all values.
// This is more efficient than loading all items when you only need a count.
// If filter is nil, counts all items with the prefix.
func CountByPrefix[T model.Model](d *DB, prefix string, newFunc func() T, filter func(T) bool) (int, error) {
	count := 0
	err := d.db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		// Only prefetch values if we need to filter
		if filter != nil {
			opts.PrefetchSize = 100
		} else {
			opts.PrefetchValues = false
		}
		it := txn.NewIterator(opts)
		defer it.Close()

		prefixBytes := []byte(prefix)
		for it.Seek(prefixBytes); it.ValidForPrefix(prefixBytes); it.Next() {
			if filter == nil {
				count++
				continue
			}
			item := it.Item()
			err := item.Value(func(val []byte) error {
				v := newFunc()
				if err := json.Unmarshal(val, v); err != nil {
					return err
				}
				if filter(v) {
					count++
				}
				return nil
			})
			if err != nil {
				return err
			}
		}
		return nil
	})
	return count, err
}
