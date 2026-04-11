package config

import (
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"
)

func TestWatcherDetectsChange(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")

	// Create initial file.
	os.WriteFile(path, []byte("version: 1"), 0o644)

	var reloadCount atomic.Int32
	w, err := NewWatcher(path, func() error {
		reloadCount.Add(1)
		return nil
	})
	if err != nil {
		t.Fatalf("NewWatcher: %v", err)
	}
	defer w.Close()

	// Modify the file.
	time.Sleep(100 * time.Millisecond) // ensure watcher is ready
	os.WriteFile(path, []byte("version: 2"), 0o644)

	// Wait for debounce (500ms) + margin.
	time.Sleep(1 * time.Second)

	if reloadCount.Load() < 1 {
		t.Fatal("expected at least 1 reload after file change")
	}
}

func TestWatcherDebounce(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	os.WriteFile(path, []byte("v: 1"), 0o644)

	var reloadCount atomic.Int32
	w, err := NewWatcher(path, func() error {
		reloadCount.Add(1)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()

	time.Sleep(100 * time.Millisecond)

	// Rapid-fire writes — should debounce to ~1 reload.
	for i := 0; i < 5; i++ {
		os.WriteFile(path, []byte(string(rune('a'+i))), 0o644)
		time.Sleep(50 * time.Millisecond)
	}

	// Wait for debounce.
	time.Sleep(1 * time.Second)

	count := reloadCount.Load()
	// Should be 1-2 (debounced), not 5.
	if count > 3 {
		t.Fatalf("expected debounced reloads (1-2), got %d", count)
	}
	if count < 1 {
		t.Fatal("expected at least 1 reload")
	}
}

func TestWatcherClose(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	os.WriteFile(path, []byte("v: 1"), 0o644)

	w, err := NewWatcher(path, func() error { return nil })
	if err != nil {
		t.Fatal(err)
	}

	// Close should not block.
	done := make(chan struct{})
	go func() {
		w.Close()
		close(done)
	}()

	select {
	case <-done:
		// Good.
	case <-time.After(2 * time.Second):
		t.Fatal("Close() blocked")
	}
}
