import React, { useEffect, useState, useRef } from "react";
import { GetTunnelLogs } from "../wailsjs/go/desktop/App";

interface LogEntry {
  timestamp: string;
  level: string;
  message: string;
}

interface LogViewerDialogProps {
  open: boolean;
  tunnelName: string;
  onClose: () => void;
}

const levelColors: Record<string, string> = {
  error: "text-bore-error",
  warn: "text-bore-warning",
  info: "text-bore-accent",
};

export const LogViewerDialog: React.FC<LogViewerDialogProps> = ({
  open,
  tunnelName,
  onClose,
}) => {
  const [entries, setEntries] = useState<LogEntry[]>([]);
  const [loading, setLoading] = useState(false);
  const scrollRef = useRef<HTMLDivElement>(null);

  const fetchLogs = async () => {
    setLoading(true);
    const result = await GetTunnelLogs(tunnelName, 200);
    setEntries(result || []);
    setLoading(false);
  };

  useEffect(() => {
    if (open && tunnelName) {
      fetchLogs();
    }
  }, [open, tunnelName]);

  useEffect(() => {
    if (scrollRef.current) {
      scrollRef.current.scrollTop = scrollRef.current.scrollHeight;
    }
  }, [entries]);

  if (!open) return null;

  return (
    <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
      <div className="bg-bore-surface border border-bore-border rounded-lg w-[600px] max-h-[70vh] flex flex-col p-4">
        <div className="flex items-center justify-between mb-3">
          <h2 className="text-sm font-bold text-bore-text">
            Logs: {tunnelName}
          </h2>
          <button
            onClick={fetchLogs}
            className="px-2 py-1 text-[10px] border border-bore-border text-bore-text-muted rounded hover:text-bore-text transition-colors"
          >
            Refresh
          </button>
        </div>

        {loading && (
          <div className="text-xs text-bore-text-muted py-8 text-center">
            Loading logs...
          </div>
        )}

        {!loading && entries.length === 0 && (
          <div className="text-xs text-bore-text-muted py-8 text-center">
            No log entries found.
          </div>
        )}

        {!loading && entries.length > 0 && (
          <div
            ref={scrollRef}
            className="flex-1 overflow-y-auto border border-bore-border rounded bg-bore-bg p-2 font-mono text-[11px] leading-relaxed"
          >
            {entries.map((entry, i) => (
              <div key={i} className="flex gap-2">
                <span className="text-bore-text-dim flex-shrink-0">
                  {entry.timestamp}
                </span>
                <span
                  className={`flex-shrink-0 w-12 ${levelColors[entry.level] || "text-bore-text-muted"}`}
                >
                  [{entry.level}]
                </span>
                <span className="text-bore-text break-all">
                  {entry.message}
                </span>
              </div>
            ))}
          </div>
        )}

        <div className="flex justify-end pt-3">
          <button
            onClick={onClose}
            className="px-3 py-1.5 text-xs border border-bore-border text-bore-text-muted rounded hover:text-bore-text transition-colors"
          >
            Close
          </button>
        </div>
      </div>
    </div>
  );
};
