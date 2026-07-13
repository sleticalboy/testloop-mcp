//go:build windows

package tools

import "os/exec"

func configureCommandProcessGroup(cmd *exec.Cmd) *exec.Cmd {
	return cmd
}
