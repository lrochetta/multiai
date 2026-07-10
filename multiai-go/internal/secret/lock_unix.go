//go:build !windows

package secret

import (
	"fmt"
	"os"
	"syscall"
)

// lockFile acquires an exclusive advisory lock on the open file.
// The lock is released by the kernel when the process exits, making it
// crash-safe: a dead process never holds a stale lock.
func lockFile(f *os.File) error {
	return syscall.Flock(int(f.Fd()), syscall.LOCK_EX)
}

// unlockFile releases the advisory lock acquired by lockFile.
func unlockFile(f *os.File) error {
	return syscall.Flock(int(f.Fd()), syscall.LOCK_UN)
}

// lockService acquires an exclusive inter-process lock for the given service.
// On Unix it uses flock(2) which auto-releases on process death (crash-safe).
func (s *encryptedFileStore) lockService(service string) (func(), error) {
	lockPath := s.lockPath(service)
	f, err := os.OpenFile(lockPath, os.O_RDWR|os.O_CREATE, 0600)
	if err != nil {
		return nil, fmt.Errorf("cannot open lock for %s: %w", service, err)
	}
	if err := lockFile(f); err != nil {
		f.Close()
		return nil, fmt.Errorf("cannot lock %s: %w", service, err)
	}
	return func() {
		_ = unlockFile(f)
		f.Close()
	}, nil
}
