CREATE TABLE IF NOT EXISTS tunnel_state (
    name              TEXT PRIMARY KEY,
    group_name        TEXT NOT NULL,
    status            TEXT NOT NULL DEFAULT 'stopped',
    local_port        INTEGER NOT NULL,
    remote_endpoint   TEXT NOT NULL,
    ssh_endpoint      TEXT,
    connected_at      DATETIME,
    disconnected_at   DATETIME,
    error_message     TEXT,
    last_error_at     DATETIME,
    retry_count       INTEGER NOT NULL DEFAULT 0,
    total_uptime_secs INTEGER NOT NULL DEFAULT 0,
    created_at        DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at        DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS connection_log (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    tunnel_name TEXT NOT NULL,
    timestamp   DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    level       TEXT NOT NULL,
    message     TEXT NOT NULL,
    FOREIGN KEY (tunnel_name) REFERENCES tunnel_state(name) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_connection_log_tunnel ON connection_log(tunnel_name, timestamp);

CREATE TABLE IF NOT EXISTS ssh_connections (
    endpoint     TEXT PRIMARY KEY,
    tunnel_count INTEGER NOT NULL DEFAULT 0,
    connected_at DATETIME,
    status       TEXT NOT NULL DEFAULT 'disconnected'
);
