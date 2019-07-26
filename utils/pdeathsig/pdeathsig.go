// +build !linux

package pdeathsig

import (
	"os/exec"

	"golang.org/x/sys/unix"
)

// Set works only on Linux.
func Set(cmd *exec.Cmd, s unix.Signal) {
	// nothing, see pdeathsig_linux.go
}
