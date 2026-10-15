//go:build !darwin && !linux && !freebsd && !openbsd && !netbsd && !windows

// Copyright Starlit Digital. Licensed under 0BSD; see LICENSE.
package toolbridge

import "errors"

func graphLock(string) (func(), error) {
	return nil, errors.New("durable graph execution is unsupported on this OS")
}
