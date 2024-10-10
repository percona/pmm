// Copyright (C) 2023 Percona LLC
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
	"time"

	"golang.org/x/sys/unix"

	"github.com/percona/pmm/utils/pdeathsig"
)

type processAction struct {
	id      string
	timeout time.Duration
	command string
	arg     []string
}

// NewProcessAction creates a new process Action.
//
// Process Action, it's an abstract Action that can run an external commands.
// This commands can be a shell script, script written on interpreted language, or binary file.
func NewProcessAction(id string, timeout time.Duration, cmd string, arg []string) Action {
	return &processAction{
		id:      id,
		timeout: timeout,
		command: cmd,
		arg:     arg,
	}
}

// ID returns an Action ID.
func (a *processAction) ID() string {
	return a.id
}

// Timeout returns Action timeout.
func (a *processAction) Timeout() time.Duration {
	return a.timeout
}

// Type returns an Action type.
func (a *processAction) Type() string {
	return a.command
}

// DSN returns a DSN for the Action.
func (a *processAction) DSN() string {
	return "" // no DSN for process action
}

// Run runs an Action and returns output and error.
func (a *processAction) Run(ctx context.Context) ([]byte, error) {
	cmd := exec.CommandContext(ctx, a.command, a.arg...) //nolint:gosec

	// restrict process
	cmd.Env = []string{} // do not inherit environment
	cmd.Dir = "/"
	pdeathsig.Set(cmd, unix.SIGKILL)

	return cmd.CombinedOutput()
}

func (*processAction) sealed() {}
