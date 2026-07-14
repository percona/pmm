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
	"fmt"
	"net"
	"os"
	"os/exec"
	"strconv"
	"time"

	"golang.org/x/sys/unix"

	"github.com/percona/pmm/api/agentpb"
	"github.com/percona/pmm/utils/pdeathsig"
)

type ptMySQLSummaryAction struct {
	id      string
	timeout time.Duration
	command string
	params  *agentpb.StartActionRequest_PTMySQLSummaryParams
}

// NewPTMySQLSummaryAction creates a new process Action.
//
// PTMySQL Summary Action, it's an abstract Action that can run an external commands.
// This commands can be a shell script, script written on interpreted language, or binary file.
func NewPTMySQLSummaryAction(id string, timeout time.Duration, cmd string, params *agentpb.StartActionRequest_PTMySQLSummaryParams) Action {
	return &ptMySQLSummaryAction{
		id:      id,
		timeout: timeout,
		command: cmd,
		params:  params,
	}
}

// ID returns an Action ID.
func (a *ptMySQLSummaryAction) ID() string {
	return a.id
}

// Timeout returns Action timeout.
func (a *ptMySQLSummaryAction) Timeout() time.Duration {
	return a.timeout
}

// Type returns an Action type.
func (a *ptMySQLSummaryAction) Type() string {
	return a.command
}

// DSN returns a DSN for the Action.
func (a *ptMySQLSummaryAction) DSN() string {
	if a.params.Socket != "" {
		return a.params.Socket
	}

	return net.JoinHostPort(a.params.Host, strconv.FormatUint(uint64(a.params.Port), 10))
}

// Run runs an Action and returns output and error.
func (a *ptMySQLSummaryAction) Run(ctx context.Context) ([]byte, error) {
	cmd := exec.CommandContext(ctx, a.command, a.ListFromMySQLParams()...) //nolint:gosec
	cmd.Env = []string{fmt.Sprintf("PATH=%s", os.Getenv("PATH"))}
	cmd.Dir = "/"
	pdeathsig.Set(cmd, unix.SIGKILL)

	return cmd.CombinedOutput()
}

// Creates an array of strings from parameters.
func (a *ptMySQLSummaryAction) ListFromMySQLParams() []string {
	if a.params == nil {
		return []string{}
	}

	var args []string
	if a.params.Socket != "" {
		args = append(args, "--socket", a.params.Socket)
	} else {
		if a.params.Host != "" {
			args = append(args, "--host", a.params.Host)
		}
		if a.params.Port > 0 && a.params.Port <= 65535 {
			args = append(args, "--port", strconv.FormatUint(uint64(a.params.Port), 10))
		}
	}

	if a.params.Username != "" {
		args = append(args, "--user", a.params.Username)
	}

	if a.params.Password != "" {
		args = append(args, "--password", a.params.Password)
	}

	return args
}

func (*ptMySQLSummaryAction) sealed() {}
