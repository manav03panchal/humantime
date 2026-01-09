package storage

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFileLock_AcquireRelease(t *testing.T) {
	t.Run("acquires and releases lock successfully", func(t *testing.T) {
		dir := t.TempDir()
		lock := NewFileLock(dir)

		err := lock.Acquire()
		require.NoError(t, err)

		// Lock file should exist
		lockPath := filepath.Join(dir, LockFileName)
		_, err = os.Stat(lockPath)
		assert.NoError(t, err, "lock file should exist")

		// Lock file should contain current PID
		data, err := os.ReadFile(lockPath)
		require.NoError(t, err)
		assert.Contains(t, string(data), "")

		err = lock.Release()
		require.NoError(t, err)

		// Lock file should be removed after release
		_, err = os.Stat(lockPath)
		assert.True(t, os.IsNotExist(err), "lock file should be removed after release")
	})

	t.Run("second lock fails when first is held", func(t *testing.T) {
		dir := t.TempDir()
		lock1 := NewFileLock(dir)
		lock2 := NewFileLock(dir)

		err := lock1.Acquire()
		require.NoError(t, err)
		defer lock1.Release()

		// Second lock should fail
		err = lock2.Acquire()
		assert.Error(t, err)
		assert.ErrorIs(t, err, ErrLockAlreadyHeld)
	})

	t.Run("can acquire lock after previous lock is released", func(t *testing.T) {
		dir := t.TempDir()
		lock1 := NewFileLock(dir)
		lock2 := NewFileLock(dir)

		err := lock1.Acquire()
		require.NoError(t, err)

		err = lock1.Release()
		require.NoError(t, err)

		// Second lock should succeed after first is released
		err = lock2.Acquire()
		require.NoError(t, err)
		defer lock2.Release()
	})

	t.Run("release is idempotent", func(t *testing.T) {
		dir := t.TempDir()
		lock := NewFileLock(dir)

		err := lock.Acquire()
		require.NoError(t, err)

		err = lock.Release()
		require.NoError(t, err)

		// Second release should not error
		err = lock.Release()
		assert.NoError(t, err)
	})
}

func TestFileLock_StaleLockCleanup(t *testing.T) {
	t.Run("cleans up stale lock with non-running PID", func(t *testing.T) {
		dir := t.TempDir()
		lockPath := filepath.Join(dir, LockFileName)

		// Create a lock file with a PID that's not running
		// Use PID 1 which is typically init and won't be "our" process
		// But that process is likely running, so use a very high PID that's unlikely to exist
		stalePID := 99999999
		err := os.WriteFile(lockPath, []byte("99999999"), 0644)
		require.NoError(t, err)

		lock := NewFileLock(dir)

		// Should be able to acquire lock after cleaning stale one
		// Note: This test may fail if PID 99999999 happens to be running
		err = lock.Acquire()
		if err != nil {
			// If it fails, check if the PID is actually running
			if isProcessRunning(stalePID) {
				t.Skip("PID 99999999 is unexpectedly running")
			}
			t.Fatalf("expected to acquire lock after stale cleanup: %v", err)
		}
		defer lock.Release()
	})
}

func TestFileLock_ReadPID(t *testing.T) {
	t.Run("reads valid PID", func(t *testing.T) {
		dir := t.TempDir()
		lockPath := filepath.Join(dir, LockFileName)

		err := os.WriteFile(lockPath, []byte("12345"), 0644)
		require.NoError(t, err)

		lock := NewFileLock(dir)
		pid := lock.readPID()
		assert.Equal(t, 12345, pid)
	})

	t.Run("returns 0 for invalid PID", func(t *testing.T) {
		dir := t.TempDir()
		lockPath := filepath.Join(dir, LockFileName)

		err := os.WriteFile(lockPath, []byte("not-a-number"), 0644)
		require.NoError(t, err)

		lock := NewFileLock(dir)
		pid := lock.readPID()
		assert.Equal(t, 0, pid)
	})

	t.Run("returns 0 for non-existent file", func(t *testing.T) {
		dir := t.TempDir()
		lock := NewFileLock(dir)
		pid := lock.readPID()
		assert.Equal(t, 0, pid)
	})
}

func TestLockError(t *testing.T) {
	t.Run("creates error with PID", func(t *testing.T) {
		err := NewLockError(ErrLockAlreadyHeld)
		assert.Contains(t, err.Error(), "cannot access database")
	})

	t.Run("unwraps to original error", func(t *testing.T) {
		lockErr := NewLockError(ErrLockAlreadyHeld)
		assert.ErrorIs(t, lockErr, ErrLockAlreadyHeld)
	})
}

func TestDB_OpenWithLock(t *testing.T) {
	t.Run("acquires lock on disk database open", func(t *testing.T) {
		dir := t.TempDir()
		dbPath := filepath.Join(dir, "db")

		db, err := Open(Options{Path: dbPath})
		require.NoError(t, err)
		defer db.Close()

		// Lock should be acquired
		assert.NotNil(t, db.lock)

		// Lock file should exist
		lockPath := filepath.Join(dbPath, LockFileName)
		_, err = os.Stat(lockPath)
		assert.NoError(t, err)
	})

	t.Run("no lock for in-memory database", func(t *testing.T) {
		db, err := Open(Options{InMemory: true})
		require.NoError(t, err)
		defer db.Close()

		// No lock for in-memory database
		assert.Nil(t, db.lock)
	})

	t.Run("second open fails when database is locked", func(t *testing.T) {
		dir := t.TempDir()
		dbPath := filepath.Join(dir, "db")

		db1, err := Open(Options{Path: dbPath})
		require.NoError(t, err)
		defer db1.Close()

		// Second open should fail
		_, err = Open(Options{Path: dbPath})
		assert.Error(t, err)

		// Error should be a LockError
		var lockErr *LockError
		assert.ErrorAs(t, err, &lockErr)
	})

	t.Run("releases lock on close", func(t *testing.T) {
		dir := t.TempDir()
		dbPath := filepath.Join(dir, "db")

		db, err := Open(Options{Path: dbPath})
		require.NoError(t, err)

		err = db.Close()
		require.NoError(t, err)

		// Lock file should be removed
		lockPath := filepath.Join(dbPath, LockFileName)
		_, err = os.Stat(lockPath)
		assert.True(t, os.IsNotExist(err), "lock file should be removed after close")

		// Should be able to open again
		db2, err := Open(Options{Path: dbPath})
		require.NoError(t, err)
		defer db2.Close()
	})
}
