// pmm-agent
// Copyright (C) 2018 Percona LLC
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

package actions

import (
	"context"
	"os/exec"

	"github.com/percona/pmm/utils/pdeathsig"
	"golang.org/x/sys/unix"
)

type processAction struct {
	id      string
	command string
	arg     []string
}

// NewProcessAction creates a new process Action.
//
// Process Action, it's an abstract Action that can run an external commands.
// This commands can be a shell script, script written on interpreted language, or binary file.
func NewProcessAction(id string, cmd string, arg []string) Action {
	return &processAction{
		id:      id,
		command: cmd,
		arg:     arg,
	}
}

// ID returns an Action ID.
func (p *processAction) ID() string {
	return p.id
}

// Type returns an Action type.
func (p *processAction) Type() string {
	return p.command
}

// Run runs an Action and returns output and error.
func (p *processAction) Run(ctx context.Context) ([]byte, error) {
	cmd := exec.CommandContext(ctx, p.command, p.arg...) //nolint:gosec

	// restrict process
	cmd.Env = []string{} // do not inherit environment
	cmd.Dir = "/"
	pdeathsig.Set(cmd, unix.SIGKILL)

	return cmd.CombinedOutput()
}

func (*processAction) sealed() {}
