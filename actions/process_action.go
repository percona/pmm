// pmm-agent
// Copyright 2019 Percona LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//  http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

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
