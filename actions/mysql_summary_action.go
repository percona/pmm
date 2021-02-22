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
	"strconv"

	"github.com/percona/pmm/api/agentpb"
	"github.com/percona/pmm/utils/pdeathsig"
	"golang.org/x/sys/unix"
)

type mysqlSummaryAction struct {
	id      string
	command string
	arg     []string
}

// NewPTMySQLSummaryAction creates a new process Action.
//
// PTMySQL Summary Action, it's an abstract Action that can run an external commands.
// This commands can be a shell script, script written on interpreted language, or binary file.
func NewPTMySQLSummaryAction(id string, cmd string, params *agentpb.StartActionRequest_PTMySQLSummaryParams) Action {
	return &mysqlSummaryAction{
		id:      id,
		command: cmd,
		arg:     argListFromMySqlParams(params),
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

// argListFromMySqlParams creates an array of strings from the pointer to the parameters for pt-mysql-sumamry
func argListFromMySqlParams(params *agentpb.StartActionRequest_PTMySQLSummaryParams) []string {
	var args []string
	if params.Socket != "" {
		args = append(args, "--socket", params.Socket)
	} else {
		if params.Host != "" {
			args = append(args, "--host", params.Host)
		}
		if params.Port > 0 && params.Port <= 65535 {
			args = append(args, "--port", strconv.FormatUint(uint64(params.Port), 10))
		}
	}

	if params.Username != "" {
		args = append(args, "--user", params.Username)
	}

	if params.Password != "" {
		args = append(args, "--password", params.Password)
	}

	return args
}
