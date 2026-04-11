package store

import (
	"path/filepath"
	"testing"
)

func newTestStore(t *testing.T) *Store {
	t.Helper()
	path := filepath.Join(t.TempDir(), "test.db")
	s, err := New(path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

func TestNewStoreCreatesMigrations(t *testing.T) {
	s := newTestStore(t)

	// Verify tables exist by inserting.
	err := s.UpsertTunnelState("test", "group", "stopped", 5432, "db:5432", "bastion:22")
	if err != nil {
		t.Fatalf("upsert: %v", err)
	}

	total, active, err := s.TunnelCount()
	if err != nil {
		t.Fatalf("count: %v", err)
	}
	if total != 1 || active != 0 {
		t.Fatalf("expected 1 total, 0 active; got %d, %d", total, active)
	}
}

func TestUpsertTunnelState(t *testing.T) {
	s := newTestStore(t)

	// Insert.
	s.UpsertTunnelState("t1", "dev", "stopped", 5432, "db:5432", "bastion:22")

	// Update.
	s.UpsertTunnelState("t1", "dev", "active", 5432, "db:5432", "bastion:22")

	total, active, _ := s.TunnelCount()
	if total != 1 || active != 1 {
		t.Fatalf("expected 1 total, 1 active; got %d, %d", total, active)
	}
}

func TestUpdateTunnelStatus(t *testing.T) {
	s := newTestStore(t)
	s.UpsertTunnelState("t1", "dev", "active", 5432, "db:5432", "bastion:22")

	err := s.UpdateTunnelStatus("t1", "error", "connection refused")
	if err != nil {
		t.Fatal(err)
	}

	_, active, _ := s.TunnelCount()
	if active != 0 {
		t.Fatalf("expected 0 active after error, got %d", active)
	}
}

func TestResetTunnelStates(t *testing.T) {
	s := newTestStore(t)
	s.UpsertTunnelState("t1", "dev", "active", 5432, "db:5432", "bastion:22")
	s.UpsertTunnelState("t2", "dev", "connecting", 6379, "redis:6379", "bastion:22")
	s.UpsertTunnelState("t3", "dev", "stopped", 8080, "web:80", "bastion:22")

	err := s.ResetTunnelStates()
	if err != nil {
		t.Fatal(err)
	}

	_, active, _ := s.TunnelCount()
	if active != 0 {
		t.Fatalf("expected 0 active after reset, got %d", active)
	}
}

func TestAppendLogAndGetLogs(t *testing.T) {
	s := newTestStore(t)
	s.UpsertTunnelState("t1", "dev", "active", 5432, "db:5432", "bastion:22")

	s.AppendLog("t1", "info", "tunnel connected")
	s.AppendLog("t1", "error", "connection lost")
	s.AppendLog("t1", "info", "reconnected")

	entries, err := s.GetLogs("t1", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 3 {
		t.Fatalf("expected 3 entries, got %d", len(entries))
	}
	// Should be chronological (oldest first).
	if entries[0].Message != "tunnel connected" {
		t.Fatalf("expected first entry 'tunnel connected', got %q", entries[0].Message)
	}
	if entries[2].Message != "reconnected" {
		t.Fatalf("expected last entry 'reconnected', got %q", entries[2].Message)
	}
}

func TestGetLogsLimit(t *testing.T) {
	s := newTestStore(t)
	s.UpsertTunnelState("t1", "dev", "active", 5432, "db:5432", "bastion:22")

	for i := 0; i < 10; i++ {
		s.AppendLog("t1", "info", "msg")
	}

	entries, _ := s.GetLogs("t1", 3)
	if len(entries) != 3 {
		t.Fatalf("expected 3 entries with limit, got %d", len(entries))
	}
}

func TestGetLogsAllTunnels(t *testing.T) {
	s := newTestStore(t)
	s.UpsertTunnelState("t1", "dev", "active", 5432, "db:5432", "bastion:22")
	s.UpsertTunnelState("t2", "dev", "active", 6379, "redis:6379", "bastion:22")

	s.AppendLog("t1", "info", "t1 msg")
	s.AppendLog("t2", "info", "t2 msg")

	entries, _ := s.GetLogs("", 10)
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries across all tunnels, got %d", len(entries))
	}
}
