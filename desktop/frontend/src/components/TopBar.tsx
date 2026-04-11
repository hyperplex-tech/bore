import React from "react";

interface TopBarProps {
  activeTunnels: number;
  connected: boolean;
}

export const TopBar: React.FC<TopBarProps> = ({ activeTunnels, connected }) => {
  return (
    <div
      className="flex items-center justify-between px-4 py-2 bg-bore-surface border-b border-bore-border"
      style={{ "--wails-draggable": "drag" } as React.CSSProperties}
    >
      <span className="text-bore-text-muted text-xs">
        Tunnels
      </span>

      <div className="flex items-center gap-3 text-xs flex-shrink-0">
        <span className="text-bore-text-muted whitespace-nowrap">
          {activeTunnels} active
        </span>
        <span className="flex items-center gap-1.5 whitespace-nowrap">
          <span
            className={`inline-block w-1.5 h-1.5 rounded-full flex-shrink-0 ${
              connected ? "bg-bore-active" : "bg-bore-error"
            }`}
          />
          <span className={connected ? "text-bore-active" : "text-bore-error"}>
            {connected ? "daemon connected" : "daemon disconnected"}
          </span>
        </span>
      </div>
    </div>
  );
};
