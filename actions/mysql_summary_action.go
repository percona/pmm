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
	"fmt"
	"os"
	"os/exec"

	"github.com/percona/pmm/utils/pdeathsig"
	"golang.org/x/sys/unix"
)

type mysqlSummaryAction struct {
	id      string
	command string
	arg     []string
}

// NewMySQLAction creates a new process Action.
//
// MySQL Action, it's an abstract Action that can run an external commands.
// This commands can be a shell script, script written on interpreted language, or binary file.
func NewMySQLAction(id string, cmd string, arg []string) Action {
	return &processAction{
		id:      id,
		command: cmd,
		arg:     arg,
	}
}

// ID returns an Action ID.
func (p *mysqlSummaryAction) ID() string {
	return p.id
}

// Type returns an Action type.
func (p *mysqlSummaryAction) Type() string {
	return p.command
}

// Run runs an Action and returns output and error.
func (p *mysqlSummaryAction) Run(ctx context.Context) ([]byte, error) {
	cmd := exec.CommandContext(ctx, p.command, p.arg...) //nolint:gosec
	cmd.Env = []string{fmt.Sprintf("PATH=%s", os.Getenv("PATH"))}
	cmd.Dir = "/"
	pdeathsig.Set(cmd, unix.SIGKILL)

	return cmd.CombinedOutput()
}

func (*mysqlSummaryAction) sealed() {}
