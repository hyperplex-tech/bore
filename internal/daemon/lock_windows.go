//go:build windows

package daemon

import (
	"fmt"
	"os"

	"golang.org/x/sys/windows"
)

// acquireLock creates an exclusive file lock using LockFileEx.
func acquireLock(path string) (*os.File, error) {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return nil, err
	}

	// LOCKFILE_EXCLUSIVE_LOCK | LOCKFILE_FAIL_IMMEDIATELY
	ol := new(windows.Overlapped)
	err = windows.LockFileEx(
		windows.Handle(f.Fd()),
		windows.LOCKFILE_EXCLUSIVE_LOCK|windows.LOCKFILE_FAIL_IMMEDIATELY,
		0, 1, 0, ol,
	)
	if err != nil {
		f.Close()
		return nil, fmt.Errorf("another bore daemon is already running (lock: %s)", path)
	}
	fmt.Fprintf(f, "%d\n", os.Getpid())
	return f, nil
}

// releaseLock releases and removes the lock file.
func releaseLock(f *os.File) {
	if f == nil {
		return
	}
	ol := new(windows.Overlapped)
	windows.UnlockFileEx(windows.Handle(f.Fd()), 0, 1, 0, ol)
	name := f.Name()
	f.Close()
	os.Remove(name)
}
