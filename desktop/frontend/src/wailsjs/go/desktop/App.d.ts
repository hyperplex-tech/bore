import { desktop } from "../models";

export function GetStatus(): Promise<desktop.StatusInfo>;
export function ListTunnels(group: string): Promise<desktop.TunnelInfo[]>;
export function ListGroups(): Promise<desktop.GroupInfo[]>;
export function ConnectTunnels(
  names: string[],
  group: string,
): Promise<desktop.TunnelInfo[]>;
export function DisconnectTunnels(
  names: string[],
  group: string,
): Promise<desktop.TunnelInfo[]>;
export function ConnectAll(): Promise<desktop.TunnelInfo[]>;
export function DisconnectAll(): Promise<desktop.TunnelInfo[]>;
export function RetryTunnel(name: string): Promise<desktop.TunnelInfo[]>;
export function PauseTunnel(name: string): Promise<desktop.TunnelInfo[]>;
export function AddTunnel(req: desktop.AddTunnelRequest): Promise<string>;
export function GetTunnelConfig(name: string): Promise<Record<string, any> | null>;
export function GetTunnelLogs(name: string, tail: number): Promise<desktop.LogEntry[]>;
export function GetAllLogs(tail: number): Promise<desktop.LogEntry[]>;
export function AddGroup(name: string, description: string): Promise<string>;
export function RenameGroup(oldName: string, newName: string): Promise<string>;
export function DeleteGroup(name: string): Promise<string>;
export function DuplicateTunnel(name: string): Promise<string>;
export function EditTunnel(req: desktop.EditTunnelRequest): Promise<string>;
export function DeleteTunnel(name: string): Promise<string>;
export function PreviewSSHImport(): Promise<desktop.SSHImportEntry[]>;
export function ImportSSHTunnels(
  names: string[],
  group: string,
): Promise<string>;
