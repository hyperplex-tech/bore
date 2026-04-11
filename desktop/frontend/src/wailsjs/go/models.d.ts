export namespace desktop {
  export interface TunnelInfo {
    name: string;
    group: string;
    type: string;
    status: string;
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

  export interface GroupInfo {
    name: string;
    description: string;
    tunnelCount: number;
    activeCount: number;
  }

  export interface StatusInfo {
    version: string;
    activeTunnels: number;
    totalTunnels: number;
    socketPath: string;
    configPath: string;
    sshAgentAvailable: boolean;
    sshAgentKeys: number;
    connected: boolean;
    tailscaleAvailable: boolean;
    tailscaleConnected: boolean;
    tailscaleIp: string;
    tailscaleHostname: string;
  }

  export interface AddTunnelRequest {
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

  export interface LogEntry {
    timestamp: string;
    level: string;
    message: string;
    tunnelName: string;
  }

  export interface EditTunnelRequest {
    originalName: string;
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

  export interface SSHImportEntry {
    name: string;
    localPort: number;
    remoteHost: string;
    remotePort: number;
    sshHost: string;
    sshUser: string;
  }
}
