//go:build windows

// Copyright Starlit Digital. Licensed under 0BSD; see LICENSE.
package toolbridge

import (
	"fmt"
	"os"
	"path/filepath"
	"syscall"
	"unsafe"
)

func graphLock(dir string) (func(), error) {
	path := filepath.Join(dir, ".lock")
	if info, e := os.Lstat(path); e == nil && !info.Mode().IsRegular() {
		return nil, fmt.Errorf("regular lock file required")
	}
	f, e := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0600)
	if e != nil {
		return nil, e
	}
	kernel := syscall.NewLazyDLL("kernel32.dll")
	lock := kernel.NewProc("LockFileEx")
	overlap := &syscall.Overlapped{}
	result, _, err := lock.Call(f.Fd(), 3, 0, 1, 0, uintptr(unsafe.Pointer(overlap)))
	if result == 0 {
		f.Close()
		return nil, fmt.Errorf("run is already locked: %w", err)
	}
	// Closing the handle releases the byte-range lock, including on process exit.
	return func() { _ = f.Close() }, nil
}
