// Copyright (C) 2024 Percona LLC
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program. If not, see <https://www.gnu.org/licenses/>.

// Package pdeathsig contains function for setting deaths singal.
package pdeathsig

import (
	"os/exec"

	"golang.org/x/sys/unix"
)

// Set sets parent-death signal s for cmd.
// See http://man7.org/linux/man-pages/man2/prctl.2.html, section PR_SET_PDEATHSIG.
func Set(cmd *exec.Cmd, s unix.Signal) {
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &unix.SysProcAttr{}
	}
	cmd.SysProcAttr.Pdeathsig = s
}
