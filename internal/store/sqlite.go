package store

import (
	"database/sql"
	_ "embed"
	"fmt"

	_ "modernc.org/sqlite"
)

//go:embed migrations/001_initial.sql
var migrationInitial string

// Store wraps a SQLite database for bore runtime state.
type Store struct {
	db *sql.DB
}

// New opens (or creates) the SQLite database at path and runs migrations.
func New(path string) (*Store, error) {
	db, err := sql.Open("sqlite", path+"?_pragma=journal_mode(wal)&_pragma=foreign_keys(on)")
	if err != nil {
		return nil, fmt.Errorf("opening database: %w", err)
	}

	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("pinging database: %w", err)
	}

	s := &Store{db: db}
	if err := s.migrate(); err != nil {
		db.Close()
		return nil, fmt.Errorf("running migrations: %w", err)
	}

	return s, nil
}

// Close closes the database.
func (s *Store) Close() error {
	return s.db.Close()
}

// DB returns the underlying *sql.DB for direct queries.
func (s *Store) DB() *sql.DB {
	return s.db
}

func (s *Store) migrate() error {
	_, err := s.db.Exec(migrationInitial)
	return err
}

// ResetTunnelStates marks all "active" or "connecting" tunnels as "stopped".
// Called on daemon startup to clean stale state from a previous crash.
func (s *Store) ResetTunnelStates() error {
	_, err := s.db.Exec(`UPDATE tunnel_state SET status = 'stopped' WHERE status IN ('active', 'connecting')`)
	return err
}

// UpsertTunnelState inserts or updates a tunnel's runtime state.
func (s *Store) UpsertTunnelState(name, group, status string, localPort int, remoteEndpoint, sshEndpoint string) error {
	_, err := s.db.Exec(`
		INSERT INTO tunnel_state (name, group_name, status, local_port, remote_endpoint, ssh_endpoint, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP)
		ON CONFLICT(name) DO UPDATE SET
			group_name = excluded.group_name,
			status = excluded.status,
			local_port = excluded.local_port,
			remote_endpoint = excluded.remote_endpoint,
			ssh_endpoint = excluded.ssh_endpoint,
			updated_at = CURRENT_TIMESTAMP
	`, name, group, status, localPort, remoteEndpoint, sshEndpoint)
	return err
}

// UpdateTunnelStatus updates just the status (and optionally error message) for a tunnel.
func (s *Store) UpdateTunnelStatus(name, status, errorMsg string) error {
	_, err := s.db.Exec(`
		UPDATE tunnel_state SET status = ?, error_message = ?, updated_at = CURRENT_TIMESTAMP
		WHERE name = ?
	`, status, errorMsg, name)
	return err
}

// AppendLog adds a log entry for a tunnel.
func (s *Store) AppendLog(tunnelName, level, message string) error {
	_, err := s.db.Exec(`
		INSERT INTO connection_log (tunnel_name, level, message) VALUES (?, ?, ?)
	`, tunnelName, level, message)
	return err
}

// TunnelCount returns total and active tunnel counts.
func (s *Store) TunnelCount() (total, active int, err error) {
	err = s.db.QueryRow(`SELECT COUNT(*) FROM tunnel_state`).Scan(&total)
	if err != nil {
		return
	}
	err = s.db.QueryRow(`SELECT COUNT(*) FROM tunnel_state WHERE status = 'active'`).Scan(&active)
	return
}

// LogEntry represents a single connection log entry.
type LogEntry struct {
	TunnelName string
	Timestamp  string
	Level      string
	Message    string
}

// GetLogs retrieves the most recent log entries for a tunnel.
// If tunnelName is empty, returns logs for all tunnels.
// limit controls how many entries to return (0 = all).
func (s *Store) GetLogs(tunnelName string, limit int) ([]LogEntry, error) {
	query := `SELECT tunnel_name, timestamp, level, message FROM connection_log`
	var args []interface{}

	if tunnelName != "" {
		query += ` WHERE tunnel_name = ?`
		args = append(args, tunnelName)
	}

	query += ` ORDER BY timestamp DESC`

	if limit > 0 {
		query += ` LIMIT ?`
		args = append(args, limit)
	}

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []LogEntry
	for rows.Next() {
		var e LogEntry
		if err := rows.Scan(&e.TunnelName, &e.Timestamp, &e.Level, &e.Message); err != nil {
			return nil, err
		}
		entries = append(entries, e)
	}

	// Reverse to chronological order (oldest first).
	for i, j := 0, len(entries)-1; i < j; i, j = i+1, j-1 {
		entries[i], entries[j] = entries[j], entries[i]
	}
	return entries, rows.Err()
}
