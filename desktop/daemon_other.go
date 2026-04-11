//go:build desktop && !windows

package desktop

import "os/exec"

// hideDaemonConsole is a no-op on non-Windows platforms.
func hideDaemonConsole(cmd *exec.Cmd) {}
