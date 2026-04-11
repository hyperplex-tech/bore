import React, { useEffect, useState } from "react";

interface SSHImportEntry {
  name: string;
  localPort: number;
  remoteHost: string;
  remotePort: number;
  sshHost: string;
  sshUser: string;
}

interface ImportDialogProps {
  open: boolean;
  onClose: () => void;
  onPreview: () => Promise<SSHImportEntry[] | null>;
  onImport: (names: string[], group: string) => void;
}

export const ImportDialog: React.FC<ImportDialogProps> = ({
  open,
  onClose,
  onPreview,
  onImport,
}) => {
  const [entries, setEntries] = useState<SSHImportEntry[]>([]);
  const [selected, setSelected] = useState<Set<string>>(new Set());
  const [group, setGroup] = useState("imported");
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");

  useEffect(() => {
    if (open) {
      setLoading(true);
      setError("");
      onPreview().then((result) => {
        setLoading(false);
        if (result && result.length > 0) {
          setEntries(result);
          setSelected(new Set(result.map((e) => e.name)));
        } else {
          setEntries([]);
          setError("No hosts with LocalForward found in ~/.ssh/config");
        }
      });
    }
  }, [open, onPreview]);

  if (!open) return null;

  const toggleEntry = (name: string) => {
    const next = new Set(selected);
    if (next.has(name)) {
      next.delete(name);
    } else {
      next.add(name);
    }
    setSelected(next);
  };

  const toggleAll = () => {
    if (selected.size === entries.length) {
      setSelected(new Set());
    } else {
      setSelected(new Set(entries.map((e) => e.name)));
    }
  };

  const handleImport = () => {
    onImport(Array.from(selected), group);
  };

  return (
    <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
      <div className="bg-bore-surface border border-bore-border rounded-lg w-[520px] max-h-[70vh] flex flex-col p-4">
        <h2 className="text-sm font-bold text-bore-text mb-3">
          Import from SSH Config
        </h2>

        {loading && (
          <div className="text-xs text-bore-text-muted py-8 text-center">
            Scanning ~/.ssh/config...
          </div>
        )}

        {error && (
          <div className="text-xs text-bore-text-muted py-8 text-center">
            {error}
          </div>
        )}

        {!loading && entries.length > 0 && (
          <>
            <div className="flex items-center justify-between mb-2">
              <button
                onClick={toggleAll}
                className="text-[10px] text-bore-accent hover:underline"
              >
                {selected.size === entries.length
                  ? "Deselect all"
                  : "Select all"}
              </button>
              <div className="flex items-center gap-2">
                <label className="text-[10px] text-bore-text-muted">
                  Import to group:
                </label>
                <input
                  className="bg-bore-bg border border-bore-border rounded px-2 py-1 text-xs text-bore-text w-28 outline-none focus:border-bore-accent"
                  value={group}
                  onChange={(e) => setGroup(e.target.value)}
                />
              </div>
            </div>

            <div className="flex-1 overflow-y-auto border border-bore-border rounded mb-3">
              {entries.map((entry) => (
                <label
                  key={entry.name}
                  className="flex items-start gap-2 px-3 py-2 hover:bg-bore-card cursor-pointer border-b border-bore-border last:border-0"
                >
                  <input
                    type="checkbox"
                    checked={selected.has(entry.name)}
                    onChange={() => toggleEntry(entry.name)}
                    className="mt-0.5 accent-bore-accent"
                  />
                  <div className="flex-1 min-w-0">
                    <div className="text-xs text-bore-text font-medium">
                      {entry.name}
                    </div>
                    <div className="text-[10px] text-bore-text-muted font-mono truncate">
                      {entry.localPort
                        ? `localhost:${entry.localPort}`
                        : "auto"}{" "}
                      &rarr; {entry.remoteHost}:{entry.remotePort} via{" "}
                      {entry.sshUser ? `${entry.sshUser}@` : ""}
                      {entry.sshHost}
                    </div>
                  </div>
                </label>
              ))}
            </div>
          </>
        )}

        <div className="flex justify-end gap-2">
          <button
            onClick={onClose}
            className="px-3 py-1.5 text-xs border border-bore-border text-bore-text-muted rounded hover:text-bore-text transition-colors"
          >
            Cancel
          </button>
          {entries.length > 0 && (
            <button
              onClick={handleImport}
              disabled={selected.size === 0}
              className="px-3 py-1.5 text-xs bg-bore-accent text-white rounded hover:bg-bore-accent-hover transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
            >
              Import {selected.size} tunnel{selected.size !== 1 ? "s" : ""}
            </button>
          )}
        </div>
      </div>
    </div>
  );
};
