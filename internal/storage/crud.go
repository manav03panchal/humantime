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
