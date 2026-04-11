import React from "react";
import type { Tunnel } from "../types/tunnel";

interface TunnelCardProps {
  tunnel: Tunnel;
  autoRefresh: boolean;
  onConnect: (name: string) => void;
  onDisconnect: (name: string) => void;
  onRetry: (name: string) => void;
  onEdit: (tunnel: Tunnel) => void;
  onDuplicate: (name: string) => void;
  onDelete: (name: string) => void;
  onViewLogs: (name: string) => void;
  onToggleAutoRefresh: (name: string) => void;
}

function formatUptime(connectedAt: string): string {
  if (!connectedAt) return "";
  const start = new Date(connectedAt).getTime();
  const now = Date.now();
  const diff = Math.floor((now - start) / 1000);
  if (diff < 60) return `${diff}s`;
  if (diff < 3600) return `${Math.floor(diff / 60)}m`;
  const h = Math.floor(diff / 3600);
  const m = Math.floor((diff % 3600) / 60);
  return `${h}h ${m}m`;
}

const statusColors: Record<string, string> = {
  active: "bg-bore-active",
  stopped: "bg-bore-stopped",
  connecting: "bg-bore-connecting",
  error: "bg-bore-error",
  retrying: "bg-bore-warning",
  paused: "bg-bore-stopped",
};

const statusBorderColors: Record<string, string> = {
  active: "border-bore-active/30",
  error: "border-bore-error/30",
  retrying: "border-bore-warning/30",
  stopped: "border-bore-border",
  connecting: "border-bore-connecting/30",
  paused: "border-bore-border",
};

export const TunnelCard: React.FC<TunnelCardProps> = ({
  tunnel,
  autoRefresh,
  onConnect,
  onDisconnect,
  onRetry,
  onEdit,
  onDuplicate,
  onDelete,
  onViewLogs,
  onToggleAutoRefresh,
}) => {
  const isActive = tunnel.status === "active";
  const isConnecting = tunnel.status === "connecting";
  const isError = tunnel.status === "error";
  const isRetrying = tunnel.status === "retrying";
  const isStopped = tunnel.status === "stopped";
  const isPaused = tunnel.status === "paused";

  const borderColor = statusBorderColors[tunnel.status] || "border-bore-border";

  return (
    <div
      className={`rounded-lg border ${borderColor} bg-bore-card px-4 py-3 mb-2 transition-colors`}
    >
      {/* Row 1: Name, badges, actions */}
      <div className="flex items-center justify-between mb-1.5">
        <div className="flex items-center gap-2">
          <span
            className={`inline-block w-2 h-2 rounded-full ${statusColors[tunnel.status] || "bg-bore-stopped"}`}
          />
          <span className="text-sm font-medium text-bore-text">
            {tunnel.name}
          </span>
          {tunnel.type !== "local" && (
            <span className="px-1.5 py-0.5 text-[10px] bg-bore-accent/20 text-bore-accent rounded">
              {tunnel.type === "k8s"
                ? "k8s"
                : tunnel.type === "dynamic"
                  ? "socks5"
                  : tunnel.type}
            </span>
          )}
          <span
            className={`px-1.5 py-0.5 text-[10px] rounded ${
              isActive
                ? "bg-bore-active/20 text-bore-active"
                : isError
                  ? "bg-bore-error/20 text-bore-error"
                  : isRetrying
                    ? "bg-bore-warning/20 text-bore-warning"
                    : "text-bore-text-muted"
            }`}
          >
            {isRetrying
              ? `error \u2014 retry in ${tunnel.nextRetrySecs}s`
              : tunnel.status}
          </span>
        </div>

        <div className="flex items-center gap-1">
          {/* Primary action */}
          {isActive && (
            <ActionButton onClick={() => onDisconnect(tunnel.name)}>
              Pause
            </ActionButton>
          )}
          {(isStopped || isPaused) && (
            <ActionButton
              variant="primary"
              onClick={() => onConnect(tunnel.name)}
            >
              Connect
            </ActionButton>
          )}
          {(isError || isRetrying) && (
            <ActionButton
              variant="primary"
              onClick={() => onRetry(tunnel.name)}
            >
              Retry
            </ActionButton>
          )}
          {isConnecting && (
            <ActionButton onClick={() => onDisconnect(tunnel.name)}>
              Cancel
            </ActionButton>
          )}

          {/* Secondary icon actions */}
          {!isConnecting && (
            <>
              <IconButton onClick={() => onEdit(tunnel)} title="Edit">
                <EditIcon />
              </IconButton>
              <IconButton onClick={() => onDuplicate(tunnel.name)} title="Duplicate">
                <DuplicateIcon />
              </IconButton>
            </>
          )}
          {(isError || isRetrying || isConnecting) && (
            <IconButton onClick={() => onViewLogs(tunnel.name)} title="Logs">
              <LogsIcon />
            </IconButton>
          )}
          {(isStopped || isError || isPaused) && (
            <IconButton onClick={() => onDelete(tunnel.name)} title="Delete" danger>
              <DeleteIcon />
            </IconButton>
          )}
        </div>
      </div>

      {/* Row 2: Connection details */}
      <div className="flex items-center gap-3 text-xs text-bore-text-muted">
        <span className="font-mono">
          {tunnel.localHost}:{tunnel.localPort}
          {tunnel.type !== "dynamic" && (
            <>
              {" "}
              <span className="text-bore-text-dim">&rarr;</span>{" "}
              {tunnel.type === "k8s"
                ? `${tunnel.k8sResource}:${tunnel.remotePort}`
                : `${tunnel.remoteHost}:${tunnel.remotePort}`}
            </>
          )}
        </span>
        <span className="text-bore-text-dim">&middot;</span>
        {tunnel.type === "k8s" ? (
          <span>
            {tunnel.k8sContext || "default"}/{tunnel.k8sNamespace || "default"}
          </span>
        ) : (
          <span>via {tunnel.sshHost}</span>
        )}
        {(isActive && tunnel.connectedAt) ||
        (isStopped && tunnel.lastErrorAt) ? (
          <>
            <span className="text-bore-text-dim">&middot;</span>
            <span>
              {isActive && tunnel.connectedAt
                ? `uptime ${formatUptime(tunnel.connectedAt)}`
                : isStopped && tunnel.lastErrorAt
                  ? `last connected ${formatUptime(tunnel.lastErrorAt)} ago`
                  : ""}
            </span>
          </>
        ) : null}
      </div>

      {/* Auto-Refresh toggle */}
      <div className="flex items-center gap-2 mt-1.5">
        <button
          onClick={() => onToggleAutoRefresh(tunnel.name)}
          className={`relative inline-flex h-4 w-7 items-center rounded-full transition-colors ${
            autoRefresh ? "bg-bore-accent" : "bg-bore-border"
          }`}
          title={autoRefresh ? "Disable auto-refresh" : "Enable auto-refresh"}
        >
          <span
            className={`inline-block h-2.5 w-2.5 rounded-full bg-white transition-transform ${
              autoRefresh ? "translate-x-3.5" : "translate-x-0.5"
            }`}
          />
        </button>
        <span className="text-[10px] text-bore-text-muted select-none">
          Auto-Refresh
        </span>
        {autoRefresh && (isError || isRetrying || isStopped || isPaused) && (
          <span className="text-[10px] text-bore-warning animate-pulse">
            reconnecting in ~5s...
          </span>
        )}
        {!autoRefresh && (
          <span className="text-[10px] text-bore-text-dim">
            Reconnects every 5s if tunnel drops
          </span>
        )}
      </div>

      {/* Error message */}
      {isError && tunnel.errorMessage && (
        <div className="mt-1.5 text-xs text-bore-error/80 truncate">
          {tunnel.errorMessage}
        </div>
      )}
    </div>
  );
};

interface ActionButtonProps {
  children: React.ReactNode;
  onClick: () => void;
  variant?: "default" | "primary";
}

const ActionButton: React.FC<ActionButtonProps> = ({
  children,
  onClick,
  variant = "default",
}) => (
  <button
    onClick={onClick}
    className={`px-2 py-1 text-[11px] rounded border transition-colors ${
      variant === "primary"
        ? "border-bore-accent text-bore-accent hover:bg-bore-accent hover:text-white"
        : "border-bore-border text-bore-text-muted hover:text-bore-text hover:border-bore-text-muted"
    }`}
  >
    {children}
  </button>
);

interface IconButtonProps {
  children: React.ReactNode;
  onClick: () => void;
  title: string;
  danger?: boolean;
}

const IconButton: React.FC<IconButtonProps> = ({
  children,
  onClick,
  title,
  danger,
}) => (
  <button
    onClick={onClick}
    title={title}
    className={`w-6 h-6 flex items-center justify-center rounded transition-colors ${
      danger
        ? "text-bore-text-muted hover:text-bore-error hover:bg-bore-error/10"
        : "text-bore-text-muted hover:text-bore-text hover:bg-bore-border/50"
    }`}
  >
    {children}
  </button>
);

const EditIcon = () => (
  <svg width="12" height="12" viewBox="0 0 12 12" fill="none" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round">
    <path d="M8.5 1.5l2 2L4 10H2v-2z" />
  </svg>
);

const DuplicateIcon = () => (
  <svg width="12" height="12" viewBox="0 0 12 12" fill="none" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round">
    <rect x="4" y="4" width="6.5" height="6.5" rx="1" />
    <path d="M8 4V2.5A1 1 0 007 1.5H2.5a1 1 0 00-1 1V7a1 1 0 001 1H4" />
  </svg>
);

const LogsIcon = () => (
  <svg width="12" height="12" viewBox="0 0 12 12" fill="none" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round">
    <path d="M3 3h6M3 6h6M3 9h4" />
  </svg>
);

const DeleteIcon = () => (
  <svg width="12" height="12" viewBox="0 0 12 12" fill="none" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round">
    <path d="M2 3h8M4.5 3V2h3v1M5 5.5v3M7 5.5v3M3.5 3l.5 7h4l.5-7" />
  </svg>
);
