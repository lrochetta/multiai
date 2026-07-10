//go:build windows

package secret

import (
	"fmt"
	"os"
	"time"
)

const (
	winLockRetries  = 50             // max retries before giving up
	winLockInterval = 100 * time.Millisecond
)

// lockService acquires an exclusive inter-process lock for the given service.
// On Windows it uses O_CREATE|O_EXCL to atomically create a lock file.
// If the file already exists (another process holds the lock), it retries
// with backoff. The lock file is removed to release the lock.
//
// NOTE: Unlike flock on Unix, this is NOT crash-safe — if the process dies
// while holding the lock, the .lock file remains on disk. The retry loop
// gives the operator a chance to notice and remove it manually. This is a
// pragmatic trade-off to avoid taking an external dependency on
// golang.org/x/sys/windows just for LockFileEx.
func (s *encryptedFileStore) lockService(service string) (func(), error) {
	lockPath := s.lockPath(service)
	for i := 0; i < winLockRetries; i++ {
		f, err := os.OpenFile(lockPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
		if err == nil {
			_, _ = fmt.Fprintf(f, "%d", os.Getpid())
			f.Close()
			return func() {
				_ = os.Remove(lockPath)
			}, nil
		}
		time.Sleep(winLockInterval)
	}
	return nil, fmt.Errorf("cannot acquire lock for %s (stale lock file? delete %s.lock and retry)", service, service)
}
