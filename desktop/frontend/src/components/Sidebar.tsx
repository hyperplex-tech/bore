import React from "react";
import type { Group } from "../types/tunnel";

interface SidebarProps {
  groups: Group[];
  selectedGroup: string | null;
  onSelectGroup: (group: string | null) => void;
  onConnectAll: () => void;
  onDisconnectAll: () => void;
  onConnectGroup: (group: string) => void;
  onDisconnectGroup: (group: string) => void;
  onNewTunnel: () => void;
  onImportSSH: () => void;
  onNewGroup: () => void;
  onRenameGroup: (oldName: string, newName: string) => void;
  onDeleteGroup: (name: string) => void;
}

export const Sidebar: React.FC<SidebarProps> = ({
  groups,
  selectedGroup,
  onSelectGroup,
  onConnectAll,
  onDisconnectAll,
  onConnectGroup,
  onDisconnectGroup,
  onNewTunnel,
  onImportSSH,
  onNewGroup,
  onRenameGroup,
  onDeleteGroup,
}) => {
  return (
    <div className="w-48 flex-shrink-0 bg-bore-sidebar border-r border-bore-border flex flex-col">
      <div
        className="px-3 pt-3 pb-2 flex items-center gap-2.5"
        style={{ "--wails-draggable": "drag" } as React.CSSProperties}
      >
        <img
          src="/icon-32.png"
          alt="Bore"
          className="w-6 h-6 flex-shrink-0"
          draggable={false}
        />
        <div className="min-w-0">
          <div className="text-bore-text font-bold text-xs tracking-wide leading-tight">Bore</div>
          <div className="text-bore-text-dim text-[9px] leading-tight">SSH Tunnel Manager</div>
        </div>
      </div>

      <div className="px-3 pt-2 pb-1">
        <div className="text-[10px] uppercase tracking-widest text-bore-text-dim mb-2">
          Groups
        </div>

        <button
          onClick={() => onSelectGroup(null)}
          className={`w-full text-left px-2 py-1.5 rounded text-xs mb-0.5 transition-colors ${
            selectedGroup === null
              ? "bg-bore-accent text-white"
              : "text-bore-text hover:bg-bore-card"
          }`}
        >
          All tunnels
        </button>

        {groups.map((g) => (
          <div key={g.name} className="group relative flex items-center mb-0.5">
            <button
              onClick={() => onSelectGroup(g.name)}
              className={`w-full text-left px-2 py-1.5 rounded text-xs flex items-center transition-colors ${
                selectedGroup === g.name
                  ? "bg-bore-accent text-white"
                  : "text-bore-text hover:bg-bore-card"
              }`}
            >
              <span className="truncate flex-1">{g.name}</span>
              {g.activeCount > 0 && (
                <span
                  className={`ml-1 px-1.5 py-0.5 rounded-full text-[10px] font-medium flex-shrink-0 ${
                    selectedGroup === g.name
                      ? "bg-white/20 text-white"
                      : "bg-bore-active/20 text-bore-active"
                  }`}
                >
                  {g.activeCount}
                </span>
              )}
            </button>
            {/* Action buttons - positioned to the right, only visible on hover */}
            <div className="hidden group-hover:flex items-center gap-0.5 absolute right-1 top-1/2 -translate-y-1/2 bg-bore-sidebar pl-1">
              <button
                onClick={() => onConnectGroup(g.name)}
                className={`w-4 h-4 flex items-center justify-center rounded text-[10px] transition-colors ${
                  selectedGroup === g.name
                    ? "hover:bg-white/20 text-bore-active"
                    : "hover:bg-bore-active/20 text-bore-active"
                }`}
                title="Connect group"
              >
                &#9654;
              </button>
              <button
                onClick={() => onDisconnectGroup(g.name)}
                className={`w-4 h-4 flex items-center justify-center rounded text-[10px] transition-colors ${
                  selectedGroup === g.name
                    ? "hover:bg-white/20 text-bore-error"
                    : "hover:bg-bore-error/20 text-bore-error"
                }`}
                title="Disconnect group"
              >
                &#9632;
              </button>
              <button
                onClick={() => {
                  const newName = prompt(`Rename group "${g.name}":`, g.name);
                  if (newName && newName !== g.name) {
                    onRenameGroup(g.name, newName);
                  }
                }}
                className={`w-4 h-4 flex items-center justify-center rounded text-[10px] transition-colors ${
                  selectedGroup === g.name
                    ? "hover:bg-white/20 text-bore-text"
                    : "hover:bg-bore-card text-bore-text-muted"
                }`}
                title="Rename group"
              >
                &#9998;
              </button>
              {g.tunnelCount === 0 && (
                <button
                  onClick={() => onDeleteGroup(g.name)}
                  className={`w-4 h-4 flex items-center justify-center rounded text-[10px] transition-colors ${
                    selectedGroup === g.name
                      ? "hover:bg-white/20 text-bore-error"
                      : "hover:bg-bore-error/20 text-bore-error"
                  }`}
                  title="Delete empty group"
                >
                  &times;
                </button>
              )}
            </div>
          </div>
        ))}
      </div>

      <div className="px-3 mt-4">
        <div className="text-[10px] uppercase tracking-widest text-bore-text-dim mb-2">
          Quick Actions
        </div>
        <button
          onClick={onNewTunnel}
          className="w-full text-left px-2 py-1.5 text-xs text-bore-text-muted hover:text-bore-text hover:bg-bore-card rounded transition-colors mb-0.5"
        >
          + New tunnel
        </button>
        <button
          onClick={onNewGroup}
          className="w-full text-left px-2 py-1.5 text-xs text-bore-text-muted hover:text-bore-text hover:bg-bore-card rounded transition-colors mb-0.5"
        >
          + New group
        </button>
        <button
          onClick={onImportSSH}
          className="w-full text-left px-2 py-1.5 text-xs text-bore-text-muted hover:text-bore-text hover:bg-bore-card rounded transition-colors mb-0.5"
        >
          + Import SSH config
        </button>
        <button
          onClick={onConnectAll}
          className="w-full text-left px-2 py-1.5 text-xs text-bore-text-muted hover:text-bore-text hover:bg-bore-card rounded transition-colors mb-0.5"
        >
          &#x21BB; Connect all
        </button>
        <button
          onClick={onDisconnectAll}
          className="w-full text-left px-2 py-1.5 text-xs text-bore-text-muted hover:text-bore-text hover:bg-bore-card rounded transition-colors"
        >
          &#x25A0; Disconnect all
        </button>
      </div>

      <div className="flex-1" />
    </div>
  );
};
