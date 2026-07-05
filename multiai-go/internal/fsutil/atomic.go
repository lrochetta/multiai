// Package fsutil holds small filesystem helpers shared across multiai.
package fsutil

import (
	"os"
	"path/filepath"
)

// WriteFileAtomic writes data to path atomically: it writes to a uniquely
// named temp file in the same directory, fsyncs and closes it, then renames it
// over the target. A crash, a concurrent writer or a disk-full (ENOSPC) can
// never leave a partially written or truncated file in place — callers either
// see the old file or the complete new one. The temp file is removed on any
// error before the rename.
//
// The unique temp name (os.CreateTemp) is what makes it safe under two
// concurrent processes writing the same target: a fixed "<path>.tmp" name
// would let them clobber each other's temp file mid-write.
func WriteFileAtomic(path string, data []byte, perm os.FileMode) error {
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, "."+filepath.Base(path)+".tmp-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	// Best-effort cleanup; a no-op once the rename below has consumed tmpName.
	defer func() { _ = os.Remove(tmpName) }()

	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Chmod(perm); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpName, path)
}
