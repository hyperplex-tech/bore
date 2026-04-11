//go:build desktop && windows

package desktop

import (
	"os/exec"
	"syscall"
)

// hideDaemonConsole sets process creation flags to prevent a console window.
func hideDaemonConsole(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{
		CreationFlags: 0x08000000, // CREATE_NO_WINDOW
	}
}
