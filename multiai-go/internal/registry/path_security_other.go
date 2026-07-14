//go:build !windows

package registry

import "os"

func isReparsePoint(os.FileInfo) bool { return false }
