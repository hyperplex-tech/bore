// @ts-check
// Wails bindings — auto-generated stubs. The real implementation is injected
// by the Wails runtime at build/dev time.

export function GetStatus() {
  return window["go"]["desktop"]["App"]["GetStatus"]();
}

export function ListTunnels(group) {
  return window["go"]["desktop"]["App"]["ListTunnels"](group);
}

export function ListGroups() {
  return window["go"]["desktop"]["App"]["ListGroups"]();
}

export function ConnectTunnels(names, group) {
  return window["go"]["desktop"]["App"]["ConnectTunnels"](names, group);
}

export function DisconnectTunnels(names, group) {
  return window["go"]["desktop"]["App"]["DisconnectTunnels"](names, group);
}

export function ConnectAll() {
  return window["go"]["desktop"]["App"]["ConnectAll"]();
}

export function DisconnectAll() {
  return window["go"]["desktop"]["App"]["DisconnectAll"]();
}

export function RetryTunnel(name) {
  return window["go"]["desktop"]["App"]["RetryTunnel"](name);
}

export function PauseTunnel(name) {
  return window["go"]["desktop"]["App"]["PauseTunnel"](name);
}

export function AddTunnel(req) {
  return window["go"]["desktop"]["App"]["AddTunnel"](req);
}

export function GetTunnelConfig(name) {
  return window["go"]["desktop"]["App"]["GetTunnelConfig"](name);
}

export function GetTunnelLogs(name, tail) {
  return window["go"]["desktop"]["App"]["GetTunnelLogs"](name, tail);
}

export function GetAllLogs(tail) {
  return window["go"]["desktop"]["App"]["GetAllLogs"](tail);
}

export function AddGroup(name, description) {
  return window["go"]["desktop"]["App"]["AddGroup"](name, description);
}

export function RenameGroup(oldName, newName) {
  return window["go"]["desktop"]["App"]["RenameGroup"](oldName, newName);
}

export function DeleteGroup(name) {
  return window["go"]["desktop"]["App"]["DeleteGroup"](name);
}

export function DuplicateTunnel(name) {
  return window["go"]["desktop"]["App"]["DuplicateTunnel"](name);
}

export function EditTunnel(req) {
  return window["go"]["desktop"]["App"]["EditTunnel"](req);
}

export function DeleteTunnel(name) {
  return window["go"]["desktop"]["App"]["DeleteTunnel"](name);
}

export function PreviewSSHImport() {
  return window["go"]["desktop"]["App"]["PreviewSSHImport"]();
}

export function ImportSSHTunnels(names, group) {
  return window["go"]["desktop"]["App"]["ImportSSHTunnels"](names, group);
}
