import React from "react";
import type { DaemonStatus } from "../types/tunnel";

interface StatusBarProps {
  status: DaemonStatus | null;
  showLogPanel?: boolean;
  onToggleLogs?: () => void;
}

export const StatusBar: React.FC<StatusBarProps> = ({ status, showLogPanel, onToggleLogs }) => {
  if (!status) {
    return (
      <div className="px-4 py-1.5 bg-bore-surface border-t border-bore-border text-[10px] text-bore-text-dim">
        Connecting to daemon...
      </div>
    );
  }

  return (
    <div className="px-4 py-1.5 bg-bore-surface border-t border-bore-border flex items-center justify-between text-[10px] text-bore-text-dim">
      <span>bore v{status.version} &mdash; daemon running</span>
      <div className="flex items-center gap-4">
        <span>
          SSH agent:{" "}
          {status.sshAgentAvailable
            ? `${status.sshAgentKeys} keys loaded`
            : "not available"}
        </span>
        {status.tailscaleAvailable && (
          <span>
            Tailscale:{" "}
            {status.tailscaleConnected
              ? status.tailscaleIp
              : "disconnected"}
          </span>
        )}
        <span>Config: {shortenPath(status.configPath)}</span>
        {onToggleLogs && (
          <button
            onClick={onToggleLogs}
            className={`px-2 py-0.5 text-[10px] border rounded transition-colors ${
              showLogPanel
                ? "border-bore-accent text-bore-accent"
                : "border-bore-border text-bore-text-dim hover:text-bore-text"
            }`}
          >
            Logs
          </button>
        )}
      </div>
    </div>
  );
};

function shortenPath(p: string): string {
  if (!p) return "";
  const home = "~";
  // Replace common home dir prefix.
  return p.replace(/^\/home\/[^/]+/, home).replace(/^\/Users\/[^/]+/, home);
}
