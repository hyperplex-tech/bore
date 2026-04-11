import React, { useEffect, useState, useCallback, useRef } from "react";
import { TopBar } from "./components/TopBar";
import { Sidebar } from "./components/Sidebar";
import { TunnelList } from "./components/TunnelList";
import { StatusBar } from "./components/StatusBar";
import { AddTunnelDialog } from "./components/AddTunnelDialog";
import { ImportDialog } from "./components/ImportDialog";
import { ConfirmDialog } from "./components/ConfirmDialog";
import { LogPanel } from "./components/LogPanel";
import type { Tunnel, Group, DaemonStatus } from "./types/tunnel";

// Wails bindings — these call Go methods on the App struct.
import {
  GetStatus,
  ListTunnels,
  ListGroups,
  ConnectTunnels,
  DisconnectTunnels,
  DisconnectAll as GoDisconnectAll,
  RetryTunnel,
  PauseTunnel,
  ConnectAll as GoConnectAll,
  AddTunnel,
  EditTunnel,
  DuplicateTunnel,
  DeleteTunnel,
  AddGroup,
  RenameGroup,
  DeleteGroup,
  PreviewSSHImport,
  ImportSSHTunnels,
} from "./wailsjs/go/desktop/App";
import { EventsOn } from "./wailsjs/runtime/runtime";

function App() {
  const [status, setStatus] = useState<DaemonStatus | null>(null);
  const [tunnels, setTunnels] = useState<Tunnel[]>([]);
  const [groups, setGroups] = useState<Group[]>([]);
  const [selectedGroup, setSelectedGroup] = useState<string | null>(null);
  const [connected, setConnected] = useState(false);
  const [showAddDialog, setShowAddDialog] = useState(false);
  const [showImportDialog, setShowImportDialog] = useState(false);
  const [editingTunnel, setEditingTunnel] = useState<Tunnel | null>(null);
  const [deletingTunnel, setDeletingTunnel] = useState<string | null>(null);
  const [showLogPanel, setShowLogPanel] = useState(false);
  const [logPanelFilter, setLogPanelFilter] = useState<string | null>(null);
  const pollRef = useRef<ReturnType<typeof setInterval>>(undefined);

  const refresh = useCallback(async () => {
    try {
      const [s, t, g] = await Promise.all([
        GetStatus(),
        ListTunnels(selectedGroup ?? ""),
        ListGroups(),
      ]);

      if (s?.connected) {
        setConnected(true);
        setStatus(s as DaemonStatus);
      } else {
        setConnected(false);
        setStatus(null);
      }

      setTunnels((t as Tunnel[]) ?? []);
      if (g) {
        const sorted = (g as Group[]).slice().sort((a, b) => a.name.localeCompare(b.name));
        setGroups(sorted);
      }
    } catch {
      setConnected(false);
    }
  }, [selectedGroup]);

  // Initial load + periodic poll.
  useEffect(() => {
    refresh();
    pollRef.current = setInterval(refresh, 3000);
    return () => clearInterval(pollRef.current);
  }, [refresh]);

  // Listen for tunnel events from the Go backend (via Wails runtime).
  useEffect(() => {
    const cleanup = EventsOn("tunnel-event", () => {
      refresh();
    });
    return cleanup;
  }, [refresh]);

  const handleConnect = useCallback(
    async (name: string) => {
      await ConnectTunnels([name], "");
      refresh();
    },
    [refresh],
  );

  const handleDisconnect = useCallback(
    async (name: string) => {
      await PauseTunnel(name);
      refresh();
    },
    [refresh],
  );

  const handleRetry = useCallback(
    async (name: string) => {
      await RetryTunnel(name);
      refresh();
    },
    [refresh],
  );

  const handleConnectAll = useCallback(async () => {
    await GoConnectAll();
    refresh();
  }, [refresh]);

  const handleDisconnectAll = useCallback(async () => {
    await GoDisconnectAll();
    refresh();
  }, [refresh]);

  const handleConnectGroup = useCallback(
    async (group: string) => {
      await ConnectTunnels([], group);
      refresh();
    },
    [refresh],
  );

  const handleDisconnectGroup = useCallback(
    async (group: string) => {
      await DisconnectTunnels([], group);
      refresh();
    },
    [refresh],
  );

  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  const handleAddTunnel = useCallback(
    async (tunnel: any) => {
      const err = await AddTunnel(tunnel);
      if (err) {
        alert(err);
        return;
      }
      setShowAddDialog(false);
      setEditingTunnel(null);
      refresh();
    },
    [refresh],
  );

  const handleEditTunnel = useCallback(
    (tunnel: Tunnel) => {
      setEditingTunnel(tunnel);
      setShowAddDialog(true);
    },
    [],
  );

  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  const handleSaveEdit = useCallback(
    async (tunnel: any) => {
      const err = await EditTunnel(tunnel);
      if (err) {
        alert(err);
        return;
      }
      setShowAddDialog(false);
      setEditingTunnel(null);
      refresh();
    },
    [refresh],
  );

  const handleDuplicate = useCallback(
    async (name: string) => {
      const err = await DuplicateTunnel(name);
      if (err) {
        alert(err);
      }
      refresh();
    },
    [refresh],
  );

  const handleDeleteTunnel = useCallback(
    async () => {
      if (!deletingTunnel) return;
      const err = await DeleteTunnel(deletingTunnel);
      if (err) {
        alert(err);
      }
      setDeletingTunnel(null);
      refresh();
    },
    [deletingTunnel, refresh],
  );

  const handleNewGroup = useCallback(async () => {
    const name = prompt("Group name:");
    if (!name) return;
    const err = await AddGroup(name, "");
    if (err) {
      alert(err);
      return;
    }
    refresh();
  }, [refresh]);

  const handleRenameGroup = useCallback(
    async (oldName: string, newName: string) => {
      const err = await RenameGroup(oldName, newName);
      if (err) {
        alert(err);
        return;
      }
      if (selectedGroup === oldName) {
        setSelectedGroup(newName);
      }
      refresh();
    },
    [refresh, selectedGroup],
  );

  const handleDeleteGroup = useCallback(
    async (name: string) => {
      const err = await DeleteGroup(name);
      if (err) {
        alert(err);
        return;
      }
      if (selectedGroup === name) {
        setSelectedGroup(null);
      }
      refresh();
    },
    [refresh, selectedGroup],
  );

  const handlePreviewImport = useCallback(async () => {
    const result = await PreviewSSHImport();
    return result;
  }, []);

  const handleImportTunnels = useCallback(
    async (names: string[], group: string) => {
      const err = await ImportSSHTunnels(names, group);
      if (err) {
        alert(err);
        return;
      }
      setShowImportDialog(false);
      refresh();
    },
    [refresh],
  );

  const activeTunnels = tunnels.filter((t) => t.status === "active").length;

  // Filter tunnels by selected group.
  const filteredTunnels = selectedGroup
    ? tunnels.filter((t) => t.group === selectedGroup)
    : tunnels;

  return (
    <div className="h-screen flex bg-bore-bg">
      <Sidebar
        groups={groups}
        selectedGroup={selectedGroup}
        onSelectGroup={setSelectedGroup}
        onConnectAll={handleConnectAll}
        onDisconnectAll={handleDisconnectAll}
        onConnectGroup={handleConnectGroup}
        onDisconnectGroup={handleDisconnectGroup}
        onNewTunnel={() => {
          setEditingTunnel(null);
          setShowAddDialog(true);
        }}
        onImportSSH={() => setShowImportDialog(true)}
        onNewGroup={handleNewGroup}
        onRenameGroup={handleRenameGroup}
        onDeleteGroup={handleDeleteGroup}
      />

      <div className="flex-1 flex flex-col min-h-0 min-w-0">
        <TopBar activeTunnels={activeTunnels} connected={connected} />

        <TunnelList
          tunnels={filteredTunnels}
          onConnect={handleConnect}
          onDisconnect={handleDisconnect}
          onRetry={handleRetry}
          onEdit={handleEditTunnel}
          onDuplicate={handleDuplicate}
          onDelete={(name) => setDeletingTunnel(name)}
          onViewLogs={(name) => {
            setLogPanelFilter(name);
            setShowLogPanel(true);
          }}
        />

        <LogPanel
          visible={showLogPanel}
          tunnelFilter={logPanelFilter}
          onClose={() => {
            setShowLogPanel(false);
            setLogPanelFilter(null);
          }}
        />

        <StatusBar
          status={status}
          showLogPanel={showLogPanel}
          onToggleLogs={() => {
            if (showLogPanel) {
              setShowLogPanel(false);
              setLogPanelFilter(null);
            } else {
              setLogPanelFilter(null);
              setShowLogPanel(true);
            }
          }}
        />
      </div>

      <AddTunnelDialog
        open={showAddDialog}
        groups={groups.map((g) => g.name)}
        editingTunnel={editingTunnel}
        onClose={() => {
          setShowAddDialog(false);
          setEditingTunnel(null);
        }}
        onAdd={handleAddTunnel}
        onEdit={handleSaveEdit}
      />

      <ImportDialog
        open={showImportDialog}
        onClose={() => setShowImportDialog(false)}
        onPreview={handlePreviewImport}
        onImport={handleImportTunnels}
      />

      <ConfirmDialog
        open={!!deletingTunnel}
        title="Delete Tunnel"
        message={`Are you sure you want to delete "${deletingTunnel}"? This cannot be undone.`}
        confirmLabel="Delete"
        danger
        onConfirm={handleDeleteTunnel}
        onCancel={() => setDeletingTunnel(null)}
      />

    </div>
  );
}

export default App;
