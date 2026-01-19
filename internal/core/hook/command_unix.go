//go:build unix

package hook

import (
	"os/exec"
	"syscall"
)

// setupProcessGroup configures the command to run in its own process group
// so that all child processes can be killed on timeout.
func setupProcessGroup(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Cancel = func() error {
		// Kill the entire process group (negative PID)
		return syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
	}
}
