//go:build !windows

package daemon

import (
	"fmt"
	"os"
	"syscall"
)

// acquireLock creates an exclusive file lock using flock.
func acquireLock(path string) (*os.File, error) {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return nil, err
	}
	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
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
	syscall.Flock(int(f.Fd()), syscall.LOCK_UN)
	name := f.Name()
	f.Close()
	os.Remove(name)
}
