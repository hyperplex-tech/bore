export type TunnelStatus =
  | "active"
  | "stopped"
  | "connecting"
  | "error"
  | "paused"
  | "retrying";

export type TunnelType = "local" | "remote" | "dynamic" | "k8s";

export interface Tunnel {
  name: string;
  group: string;
  type: TunnelType;
  status: TunnelStatus;
  localPort: number;
  localHost: string;
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
  errorMessage: string;
  connectedAt: string;
  lastErrorAt: string;
  retryCount: number;
  nextRetrySecs: number;
}

export interface Group {
  name: string;
  description: string;
  tunnelCount: number;
  activeCount: number;
}

export interface DaemonStatus {
  version: string;
  activeTunnels: number;
  totalTunnels: number;
  socketPath: string;
  configPath: string;
  sshAgentAvailable: boolean;
  sshAgentKeys: number;
  tailscaleAvailable: boolean;
  tailscaleConnected: boolean;
  tailscaleIp: string;
  tailscaleHostname: string;
}

export interface LogEntry {
  timestamp: string;
  level: string;
  message: string;
}
