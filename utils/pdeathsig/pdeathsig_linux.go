// Copyright (C) 2023 Percona LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

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
