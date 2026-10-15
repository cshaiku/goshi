//go:build darwin || linux || freebsd || openbsd || netbsd

// Copyright Starlit Digital. Licensed under 0BSD; see LICENSE.
package toolbridge

import (
	"fmt"
	"os"
	"path/filepath"
	"syscall"
)

// Kernel locks release on process exit, including an interrupted checkpoint writer.
func graphLock(dir string) (func(), error) {
	path := filepath.Join(dir, ".lock")
	if info, e := os.Lstat(path); e == nil && !info.Mode().IsRegular() {
		return nil, fmt.Errorf("regular lock file required")
	}
	f, e := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0600)
	if e != nil {
		return nil, e
	}
	if e = syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); e != nil {
		f.Close()
		return nil, fmt.Errorf("run is already locked: %w", e)
	}
	return func() { _ = syscall.Flock(int(f.Fd()), syscall.LOCK_UN); _ = f.Close() }, nil
}
