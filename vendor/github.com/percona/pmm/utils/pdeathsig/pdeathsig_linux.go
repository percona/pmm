package pdeathsig

import (
	"os/exec"

	"golang.org/x/sys/unix"
)

// Set sets parent-death signal s for cmd.
// See http://man7.org/linux/man-pages/man2/prctl.2.html, section PR_SET_PDEATHSIG.
func Set(cmd *exec.Cmd, s unix.Signal) {
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = new(unix.SysProcAttr)
	}
	cmd.SysProcAttr.Pdeathsig = s
}
