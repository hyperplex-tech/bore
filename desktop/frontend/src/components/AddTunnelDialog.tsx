import React, { useState, useEffect, useRef } from "react";
import type { Tunnel } from "../types/tunnel";
import { GetTunnelConfig } from "../wailsjs/go/desktop/App";

type TunnelTypeOption = "local" | "remote" | "dynamic" | "k8s";

interface TunnelFormData {
  name: string;
  group: string;
  type: string;
  localHost: string;
  localPort: number;
  remoteHost: string;
  remotePort: number;
  sshHost: string;
  sshPort: number;
  sshUser: string;
  authMethod: string;
  identityFile: string;
  jumpHosts: string[];
  k8sContext: string;
  k8sNamespace: string;
  k8sResource: string;
  preConnect: string;
  postConnect: string;
  reconnect: boolean | null;
}

interface AddTunnelDialogProps {
  open: boolean;
  groups: string[];
  editingTunnel?: Tunnel | null;
  onClose: () => void;
  onAdd: (tunnel: TunnelFormData) => void;
  onEdit?: (tunnel: TunnelFormData & { originalName: string }) => void;
}

const tunnelTypes: { value: TunnelTypeOption; label: string; desc: string }[] = [
  { value: "local", label: "Local Forward", desc: "SSH local port forward" },
  { value: "remote", label: "Remote Forward", desc: "SSH reverse tunnel" },
  { value: "dynamic", label: "SOCKS5 Proxy", desc: "SSH dynamic forward" },
  { value: "k8s", label: "K8s Port-Forward", desc: "kubectl port-forward" },
];

export const AddTunnelDialog: React.FC<AddTunnelDialogProps> = ({
  open,
  groups,
  editingTunnel,
  onClose,
  onAdd,
  onEdit,
}) => {
  const [tunnelType, setTunnelType] = useState<TunnelTypeOption>("local");
  const [groupOpen, setGroupOpen] = useState(false);
  const groupRef = useRef<HTMLDivElement>(null);
  const [name, setName] = useState("");
  const [group, setGroup] = useState(groups[0] || "default");
  const [localHost, setLocalHost] = useState("127.0.0.1");
  const [localPort, setLocalPort] = useState("");
  const [remoteHost, setRemoteHost] = useState("");
  const [remotePort, setRemotePort] = useState("");
  const [sshHost, setSshHost] = useState("");
  const [sshPort, setSshPort] = useState("22");
  const [sshUser, setSshUser] = useState("");
  const [authMethod, setAuthMethod] = useState("agent");
  const [identityFile, setIdentityFile] = useState("");
  const [jumpHosts, setJumpHosts] = useState("");
  const [k8sContext, setK8sContext] = useState("");
  const [k8sNamespace, setK8sNamespace] = useState("default");
  const [k8sResource, setK8sResource] = useState("");
  const [preConnect, setPreConnect] = useState("");
  const [postConnect, setPostConnect] = useState("");
  const [reconnect, setReconnect] = useState<"default" | "yes" | "no">("default");
  const [showAdvanced, setShowAdvanced] = useState(false);
  const [error, setError] = useState("");

  const isEditing = !!editingTunnel;

  // Close group dropdown on outside click.
  useEffect(() => {
    const handler = (e: MouseEvent) => {
      if (groupRef.current && !groupRef.current.contains(e.target as Node)) {
        setGroupOpen(false);
      }
    };
    document.addEventListener("mousedown", handler);
    return () => document.removeEventListener("mousedown", handler);
  }, []);

  // Pre-fill fields when editing.
  useEffect(() => {
    if (editingTunnel) {
      setTunnelType((editingTunnel.type || "local") as TunnelTypeOption);
      setName(editingTunnel.name);
      setGroup(editingTunnel.group);
      setLocalHost(editingTunnel.localHost || "127.0.0.1");
      setLocalPort(String(editingTunnel.localPort || ""));
      setRemoteHost(editingTunnel.remoteHost || "");
      setRemotePort(String(editingTunnel.remotePort || ""));
      setSshHost(editingTunnel.sshHost || "");
      setSshPort(String(editingTunnel.sshPort || 22));
      setSshUser(editingTunnel.sshUser || "");
      setAuthMethod(editingTunnel.authMethod || "agent");
      setIdentityFile(editingTunnel.identityFile || "");
      setJumpHosts((editingTunnel.jumpHosts || []).join(", "));
      setK8sContext(editingTunnel.k8sContext || "");
      setK8sNamespace(editingTunnel.k8sNamespace || "default");
      setK8sResource(editingTunnel.k8sResource || "");

      // Load config-only fields (hooks, reconnect) from config.
      GetTunnelConfig(editingTunnel.name).then((cfg) => {
        if (cfg) {
          setPreConnect((cfg.preConnect as string) || "");
          setPostConnect((cfg.postConnect as string) || "");
          if (cfg.reconnect === true) setReconnect("yes");
          else if (cfg.reconnect === false) setReconnect("no");
          else setReconnect("default");
        }
      }).catch(() => {
        // Silently fall back to defaults if config can't be loaded.
      });

      // Show advanced if any advanced fields are set.
      if (
        editingTunnel.identityFile ||
        (editingTunnel.jumpHosts && editingTunnel.jumpHosts.length > 0) ||
        editingTunnel.localHost !== "127.0.0.1"
      ) {
        setShowAdvanced(true);
      }
    } else {
      setTunnelType("local");
      setName("");
      setGroup(groups[0] || "default");
      setLocalHost("127.0.0.1");
      setLocalPort("");
      setRemoteHost("");
      setRemotePort("");
      setSshHost("");
      setSshPort("22");
      setSshUser("");
      setAuthMethod("agent");
      setIdentityFile("");
      setJumpHosts("");
      setK8sContext("");
      setK8sNamespace("default");
      setK8sResource("");
      setPreConnect("");
      setPostConnect("");
      setReconnect("default");
      setShowAdvanced(false);
    }
    setError("");
  // Note: `groups` is intentionally excluded — it changes on every poll and
  // would reset the form while the user is typing.
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [editingTunnel, open]);

  if (!open) return null;

  const isDynamic = tunnelType === "dynamic";
  const isK8s = tunnelType === "k8s";
  const needsRemote = tunnelType === "local" || tunnelType === "remote";

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    setError("");

    if (!name || !localPort) {
      setError("Name and local port are required.");
      return;
    }

    if (isK8s) {
      if (!k8sResource || !remotePort) {
        setError("K8s resource and remote port are required.");
        return;
      }
    } else if (isDynamic) {
      if (!sshHost) {
        setError("SSH host is required for SOCKS5 proxy.");
        return;
      }
    } else {
      if (!remoteHost || !remotePort || !sshHost) {
        setError("Remote host, remote port, and SSH host are required.");
        return;
      }
    }

    const parsedJumpHosts = jumpHosts
      .split(",")
      .map((h) => h.trim())
      .filter(Boolean);

    let reconnectVal: boolean | null = null;
    if (reconnect === "yes") reconnectVal = true;
    else if (reconnect === "no") reconnectVal = false;

    const tunnelData: TunnelFormData = {
      name,
      group,
      type: tunnelType,
      localHost,
      localPort: parseInt(localPort, 10),
      remoteHost,
      remotePort: parseInt(remotePort, 10) || 0,
      sshHost,
      sshPort: parseInt(sshPort, 10) || 22,
      sshUser,
      authMethod,
      identityFile,
      jumpHosts: parsedJumpHosts,
      k8sContext,
      k8sNamespace,
      k8sResource,
      preConnect,
      postConnect,
      reconnect: reconnectVal,
    };

    if (isEditing && onEdit) {
      onEdit({ ...tunnelData, originalName: editingTunnel!.name });
    } else {
      onAdd(tunnelData);
    }
  };

  const inputClass =
    "w-full bg-bore-bg border border-bore-border rounded px-2 py-1.5 text-xs text-bore-text outline-none focus:border-bore-accent";

  return (
    <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
      <div className="bg-bore-surface border border-bore-border rounded-lg w-[460px] max-h-[80vh] overflow-y-auto p-4">
        <h2 className="text-sm font-bold text-bore-text mb-3">
          {isEditing ? "Edit Tunnel" : "New Tunnel"}
        </h2>

        <form onSubmit={handleSubmit} className="space-y-2">
          {/* Type selector */}
          <div className="grid grid-cols-4 gap-1 mb-1">
            {tunnelTypes.map((t) => (
              <button
                key={t.value}
                type="button"
                onClick={() => setTunnelType(t.value)}
                className={`px-1.5 py-1.5 text-[10px] rounded border transition-colors text-center ${
                  tunnelType === t.value
                    ? "border-bore-accent bg-bore-accent/10 text-bore-accent"
                    : "border-bore-border text-bore-text-muted hover:text-bore-text"
                }`}
                title={t.desc}
              >
                {t.label}
              </button>
            ))}
          </div>

          {/* Name + Group */}
          <div className="flex gap-2">
            <div className="flex-1">
              <label className="text-[10px] text-bore-text-muted block mb-0.5">
                Name
              </label>
              <input
                className={inputClass}
                value={name}
                onChange={(e) => setName(e.target.value)}
                placeholder="dev-postgresql"
              />
            </div>
            <div className="w-32 relative" ref={groupRef}>
              <label className="text-[10px] text-bore-text-muted block mb-0.5">
                Group
              </label>
              <input
                className={inputClass + " pr-5"}
                value={group}
                onChange={(e) => {
                  setGroup(e.target.value);
                  setGroupOpen(true);
                }}
                onFocus={() => setGroupOpen(true)}
                placeholder="default"
                style={{
                  backgroundImage: `url("data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' width='12' height='12' viewBox='0 0 12 12'%3E%3Cpath fill='%23888' d='M3 5l3 3 3-3'/%3E%3C/svg%3E")`,
                  backgroundRepeat: "no-repeat",
                  backgroundPosition: "right 6px center",
                }}
              />
              {groupOpen && groups.length > 0 && (
                <div className="absolute z-10 mt-0.5 w-full bg-bore-surface border border-bore-border rounded shadow-lg max-h-32 overflow-y-auto">
                  {groups.map((g) => (
                    <button
                      key={g}
                      type="button"
                      onClick={() => {
                        setGroup(g);
                        setGroupOpen(false);
                      }}
                      className={`w-full text-left px-2 py-1 text-xs hover:bg-bore-accent/20 ${
                        g === group ? "text-bore-accent" : "text-bore-text"
                      }`}
                    >
                      {g}
                    </button>
                  ))}
                </div>
              )}
            </div>
          </div>

          {/* K8s fields */}
          {isK8s && (
            <>
              <div>
                <label className="text-[10px] text-bore-text-muted block mb-0.5">
                  Resource
                </label>
                <input
                  className={inputClass}
                  value={k8sResource}
                  onChange={(e) => setK8sResource(e.target.value)}
                  placeholder="svc/my-service or pod/my-pod"
                />
              </div>
              <div className="flex gap-2">
                <div className="w-24">
                  <label className="text-[10px] text-bore-text-muted block mb-0.5">
                    Local Port
                  </label>
                  <input
                    className={inputClass}
                    type="number"
                    value={localPort}
                    onChange={(e) => setLocalPort(e.target.value)}
                    placeholder="8080"
                  />
                </div>
                <div className="w-24">
                  <label className="text-[10px] text-bore-text-muted block mb-0.5">
                    Remote Port
                  </label>
                  <input
                    className={inputClass}
                    type="number"
                    value={remotePort}
                    onChange={(e) => setRemotePort(e.target.value)}
                    placeholder="8080"
                  />
                </div>
                <div className="flex-1">
                  <label className="text-[10px] text-bore-text-muted block mb-0.5">
                    Context
                  </label>
                  <input
                    className={inputClass}
                    value={k8sContext}
                    onChange={(e) => setK8sContext(e.target.value)}
                    placeholder="(optional)"
                  />
                </div>
              </div>
              <div>
                <label className="text-[10px] text-bore-text-muted block mb-0.5">
                  Namespace
                </label>
                <input
                  className={inputClass}
                  value={k8sNamespace}
                  onChange={(e) => setK8sNamespace(e.target.value)}
                  placeholder="default"
                />
              </div>
            </>
          )}

          {/* SSH-based tunnel fields (local, remote, dynamic) */}
          {!isK8s && (
            <>
              {/* Port + remote target */}
              <div className="flex gap-2">
                <div className="w-24">
                  <label className="text-[10px] text-bore-text-muted block mb-0.5">
                    Local Port
                  </label>
                  <input
                    className={inputClass}
                    type="number"
                    value={localPort}
                    onChange={(e) => setLocalPort(e.target.value)}
                    placeholder={isDynamic ? "1080" : "5432"}
                  />
                </div>
                {needsRemote && (
                  <>
                    <div className="flex-1">
                      <label className="text-[10px] text-bore-text-muted block mb-0.5">
                        Remote Host
                      </label>
                      <input
                        className={inputClass}
                        value={remoteHost}
                        onChange={(e) => setRemoteHost(e.target.value)}
                        placeholder="db.dev.internal"
                      />
                    </div>
                    <div className="w-24">
                      <label className="text-[10px] text-bore-text-muted block mb-0.5">
                        Remote Port
                      </label>
                      <input
                        className={inputClass}
                        type="number"
                        value={remotePort}
                        onChange={(e) => setRemotePort(e.target.value)}
                        placeholder="5432"
                      />
                    </div>
                  </>
                )}
              </div>

              {/* SSH connection */}
              <div className="flex gap-2">
                <div className="flex-1">
                  <label className="text-[10px] text-bore-text-muted block mb-0.5">
                    SSH Host (via)
                  </label>
                  <input
                    className={inputClass}
                    value={sshHost}
                    onChange={(e) => setSshHost(e.target.value)}
                    placeholder="bastion.example.com"
                  />
                </div>
                <div className="w-20">
                  <label className="text-[10px] text-bore-text-muted block mb-0.5">
                    SSH Port
                  </label>
                  <input
                    className={inputClass}
                    type="number"
                    value={sshPort}
                    onChange={(e) => setSshPort(e.target.value)}
                  />
                </div>
                <div className="w-24">
                  <label className="text-[10px] text-bore-text-muted block mb-0.5">
                    SSH User
                  </label>
                  <input
                    className={inputClass}
                    value={sshUser}
                    onChange={(e) => setSshUser(e.target.value)}
                    placeholder="deploy"
                  />
                </div>
              </div>
            </>
          )}

          {/* Advanced section (collapsible) */}
          <div>
            <button
              type="button"
              onClick={() => setShowAdvanced(!showAdvanced)}
              className="text-[10px] text-bore-accent hover:underline"
            >
              {showAdvanced ? "Hide advanced" : "Show advanced"}
            </button>
          </div>

          {showAdvanced && (
            <div className="space-y-2 border-t border-bore-border pt-2">
              {/* Auth method + identity file (SSH-based only) */}
              {!isK8s && (
                <div className="flex gap-2">
                  <div className="w-28">
                    <label className="text-[10px] text-bore-text-muted block mb-0.5">
                      Auth Method
                    </label>
                    <select
                      className={inputClass}
                      value={authMethod}
                      onChange={(e) => setAuthMethod(e.target.value)}
                    >
                      <option value="agent">SSH Agent</option>
                      <option value="key">Private Key</option>
                    </select>
                  </div>
                  <div className="flex-1">
                    <label className="text-[10px] text-bore-text-muted block mb-0.5">
                      Identity File
                    </label>
                    <input
                      className={inputClass}
                      value={identityFile}
                      onChange={(e) => setIdentityFile(e.target.value)}
                      placeholder="~/.ssh/id_rsa"
                    />
                  </div>
                </div>
              )}

              {/* Jump hosts (SSH-based only) */}
              {!isK8s && (
                <div>
                  <label className="text-[10px] text-bore-text-muted block mb-0.5">
                    Jump Hosts (comma-separated)
                  </label>
                  <input
                    className={inputClass}
                    value={jumpHosts}
                    onChange={(e) => setJumpHosts(e.target.value)}
                    placeholder="jump1.example.com, jump2.example.com"
                  />
                </div>
              )}

              {/* Local host binding */}
              <div className="flex gap-2">
                <div className="w-40">
                  <label className="text-[10px] text-bore-text-muted block mb-0.5">
                    Local Host
                  </label>
                  <input
                    className={inputClass}
                    value={localHost}
                    onChange={(e) => setLocalHost(e.target.value)}
                    placeholder="127.0.0.1"
                  />
                </div>
                <div className="w-28">
                  <label className="text-[10px] text-bore-text-muted block mb-0.5">
                    Auto-Reconnect
                  </label>
                  <select
                    className={inputClass}
                    value={reconnect}
                    onChange={(e) =>
                      setReconnect(
                        e.target.value as "default" | "yes" | "no",
                      )
                    }
                  >
                    <option value="default">Default</option>
                    <option value="yes">Yes</option>
                    <option value="no">No</option>
                  </select>
                </div>
              </div>

              {/* Hooks */}
              <div>
                <label className="text-[10px] text-bore-text-muted block mb-0.5">
                  Pre-Connect Hook
                </label>
                <input
                  className={inputClass}
                  value={preConnect}
                  onChange={(e) => setPreConnect(e.target.value)}
                  placeholder="echo 'connecting...'"
                />
              </div>
              <div>
                <label className="text-[10px] text-bore-text-muted block mb-0.5">
                  Post-Connect Hook
                </label>
                <input
                  className={inputClass}
                  value={postConnect}
                  onChange={(e) => setPostConnect(e.target.value)}
                  placeholder="notify-send 'tunnel up'"
                />
              </div>
            </div>
          )}

          {error && (
            <div className="text-xs text-bore-error">{error}</div>
          )}

          <div className="flex justify-end gap-2 pt-2">
            <button
              type="button"
              onClick={onClose}
              className="px-3 py-1.5 text-xs border border-bore-border text-bore-text-muted rounded hover:text-bore-text transition-colors"
            >
              Cancel
            </button>
            <button
              type="submit"
              className="px-3 py-1.5 text-xs bg-bore-accent text-white rounded hover:bg-bore-accent-hover transition-colors"
            >
              {isEditing ? "Save" : "Add Tunnel"}
            </button>
          </div>
        </form>
      </div>
    </div>
  );
};
