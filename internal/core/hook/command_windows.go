//go:build windows

package hook

import (
	"os/exec"
)

// setupProcessGroup on Windows is a no-op.
// Windows handles process termination differently, and exec.CommandContext
// already terminates the process on context cancellation.
func setupProcessGroup(cmd *exec.Cmd) {
	// No-op on Windows
	_ = cmd
}
