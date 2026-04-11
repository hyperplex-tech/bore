import React, { useEffect, useState, useRef, useCallback } from "react";
import { GetAllLogs, GetTunnelLogs } from "../wailsjs/go/desktop/App";

interface LogEntry {
  timestamp: string;
  level: string;
  message: string;
  tunnelName: string;
}

interface LogPanelProps {
  visible: boolean;
  onClose: () => void;
  /** If set, filter logs to this tunnel only. */
  tunnelFilter?: string | null;
}

const levelColors: Record<string, string> = {
  error: "text-bore-error",
  warn: "text-bore-warning",
  info: "text-bore-accent",
};

export const LogPanel: React.FC<LogPanelProps> = ({
  visible,
  onClose,
  tunnelFilter,
}) => {
  const [entries, setEntries] = useState<LogEntry[]>([]);
  const [autoRefresh, setAutoRefresh] = useState(true);
  const scrollRef = useRef<HTMLDivElement>(null);
  const intervalRef = useRef<ReturnType<typeof setInterval>>(undefined);

  const fetchLogs = useCallback(async () => {
    try {
      let result: LogEntry[];
      if (tunnelFilter) {
        result = await GetTunnelLogs(tunnelFilter, 200);
      } else {
        result = await GetAllLogs(200);
      }
      setEntries(result || []);
    } catch {
      // ignore fetch errors
    }
  }, [tunnelFilter]);

  // Fetch on open and when filter changes.
  useEffect(() => {
    if (visible) {
      fetchLogs();
    }
  }, [visible, fetchLogs]);

  // Auto-refresh every 3s when enabled.
  useEffect(() => {
    if (visible && autoRefresh) {
      intervalRef.current = setInterval(fetchLogs, 3000);
      return () => clearInterval(intervalRef.current);
    }
    return () => clearInterval(intervalRef.current);
  }, [visible, autoRefresh, fetchLogs]);

  // Auto-scroll to bottom on new entries.
  useEffect(() => {
    if (scrollRef.current) {
      scrollRef.current.scrollTop = scrollRef.current.scrollHeight;
    }
  }, [entries]);

  if (!visible) return null;

  return (
    <div className="flex flex-col border-t border-bore-border bg-bore-surface" style={{ height: "30vh", minHeight: 120 }}>
      {/* Header */}
      <div className="flex items-center justify-between px-3 py-1.5 border-b border-bore-border bg-bore-bg/50">
        <div className="flex items-center gap-2">
          <span className="text-[10px] font-bold text-bore-text uppercase tracking-wider">
            Logs
          </span>
          {tunnelFilter && (
            <span className="text-[10px] text-bore-accent font-mono">
              {tunnelFilter}
            </span>
          )}
        </div>
        <div className="flex items-center gap-2">
          <button
            onClick={() => setAutoRefresh(!autoRefresh)}
            className={`px-2 py-0.5 text-[10px] border rounded transition-colors ${
              autoRefresh
                ? "border-bore-accent text-bore-accent"
                : "border-bore-border text-bore-text-muted hover:text-bore-text"
            }`}
            title={autoRefresh ? "Disable auto-refresh" : "Enable auto-refresh"}
          >
            {autoRefresh ? "Auto" : "Paused"}
          </button>
          <button
            onClick={fetchLogs}
            className="px-2 py-0.5 text-[10px] border border-bore-border text-bore-text-muted rounded hover:text-bore-text transition-colors"
          >
            Refresh
          </button>
          <button
            onClick={onClose}
            className="px-2 py-0.5 text-[10px] border border-bore-border text-bore-text-muted rounded hover:text-bore-text transition-colors"
          >
            Close
          </button>
        </div>
      </div>

      {/* Log entries */}
      {entries.length === 0 ? (
        <div className="flex-1 flex items-center justify-center text-xs text-bore-text-muted">
          No log entries found.
        </div>
      ) : (
        <div
          ref={scrollRef}
          className="flex-1 overflow-y-auto p-2 font-mono text-[11px] leading-relaxed"
        >
          {entries.map((entry, i) => (
            <div key={i} className="flex gap-2 hover:bg-bore-bg/30">
              <span className="text-bore-text-dim flex-shrink-0">
                {entry.timestamp}
              </span>
              <span
                className={`flex-shrink-0 w-12 ${levelColors[entry.level] || "text-bore-text-muted"}`}
              >
                [{entry.level}]
              </span>
              {!tunnelFilter && entry.tunnelName && (
                <span className="flex-shrink-0 text-bore-text-muted w-24 truncate" title={entry.tunnelName}>
                  {entry.tunnelName}
                </span>
              )}
              <span className="text-bore-text break-all">
                {entry.message}
              </span>
            </div>
          ))}
        </div>
      )}
    </div>
  );
};
