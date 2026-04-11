import React, { useState } from "react";
import type { Tunnel, TunnelStatus } from "../types/tunnel";
import { TunnelCard } from "./TunnelCard";

interface TunnelListProps {
  tunnels: Tunnel[];
  onConnect: (name: string) => void;
  onDisconnect: (name: string) => void;
  onRetry: (name: string) => void;
  onEdit: (tunnel: Tunnel) => void;
  onDuplicate: (name: string) => void;
  onDelete: (name: string) => void;
  onViewLogs: (name: string) => void;
}

const statusFilters: { label: string; value: TunnelStatus | "all" }[] = [
  { label: "All status", value: "all" },
  { label: "Active", value: "active" },
  { label: "Stopped", value: "stopped" },
  { label: "Paused", value: "paused" },
  { label: "Error", value: "error" },
  { label: "Connecting", value: "connecting" },
];

export const TunnelList: React.FC<TunnelListProps> = ({
  tunnels,
  onConnect,
  onDisconnect,
  onRetry,
  onEdit,
  onDuplicate,
  onDelete,
  onViewLogs,
}) => {
  const [statusFilter, setStatusFilter] = useState<TunnelStatus | "all">(
    "all",
  );

  const filtered =
    statusFilter === "all"
      ? tunnels
      : tunnels.filter((t) => t.status === statusFilter);

  // Sort: active first, then error/retrying, then stopped.
  const statusOrder: Record<string, number> = {
    active: 0,
    connecting: 1,
    retrying: 2,
    error: 3,
    paused: 4,
    stopped: 5,
  };
  const sorted = [...filtered].sort((a, b) => {
    const so = (statusOrder[a.status] ?? 9) - (statusOrder[b.status] ?? 9);
    if (so !== 0) return so;
    return a.name.localeCompare(b.name);
  });

  return (
    <div className="flex-1 flex flex-col min-h-0">
      <div className="px-4 py-2 max-w-3xl">
        <select
          value={statusFilter}
          onChange={(e) =>
            setStatusFilter(e.target.value as TunnelStatus | "all")
          }
          className="bg-bore-card border border-bore-border rounded px-2 py-1.5 pr-5 text-xs text-bore-text outline-none focus:border-bore-accent cursor-pointer"
          style={{
            WebkitAppearance: "none",
            appearance: "none",
            backgroundImage: `url("data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' width='12' height='12' viewBox='0 0 12 12'%3E%3Cpath fill='%23888' d='M3 5l3 3 3-3'/%3E%3C/svg%3E")`,
            backgroundRepeat: "no-repeat",
            backgroundPosition: "right 6px center",
          }}
        >
          {statusFilters.map((f) => (
            <option key={f.value} value={f.value}>
              {f.label}
            </option>
          ))}
        </select>
      </div>

      <div className="flex-1 overflow-y-auto px-4 pb-2 max-w-3xl">
        {sorted.length === 0 ? (
          <div className="flex flex-col items-center justify-center py-16 text-center">
            <img
              src="/icon-128.png"
              alt="Bore"
              className="w-20 h-20 opacity-30 mb-4"
              draggable={false}
            />
            <div className="text-bore-text-muted text-sm mb-1">
              {tunnels.length === 0
                ? "No tunnels configured"
                : "No tunnels match the filter"}
            </div>
            {tunnels.length === 0 && (
              <div className="text-bore-text-dim text-xs">
                Create a tunnel or import from your SSH config to get started.
              </div>
            )}
          </div>
        ) : (
          sorted.map((t) => (
            <TunnelCard
              key={t.name}
              tunnel={t}
              onConnect={onConnect}
              onDisconnect={onDisconnect}
              onRetry={onRetry}
              onEdit={onEdit}
              onDuplicate={onDuplicate}
              onDelete={onDelete}
              onViewLogs={onViewLogs}
            />
          ))
        )}
      </div>
    </div>
  );
};
