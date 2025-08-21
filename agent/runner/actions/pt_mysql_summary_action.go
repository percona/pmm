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
	"bytes"
	"context"
	"errors"
	"fmt"
	"html/template"
	"net"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"golang.org/x/sys/unix"

	agentv1 "github.com/percona/pmm/api/agent/v1"
	"github.com/percona/pmm/utils/pdeathsig"
)

type ptMySQLSummaryAction struct {
	id      string
	timeout time.Duration
	command string
	params  *agentv1.StartActionRequest_PTMySQLSummaryParams
}

// ErrInvalidCharacter is returned when a parameter contains invalid characters.
var ErrInvalidCharacter = errors.New("parameter contains invalid character(s)")

// NewPTMySQLSummaryAction creates a new process Action.
//
// PTMySQL Summary Action, it's an abstract Action that can run an external commands.
// This commands can be a shell script, script written on interpreted language, or binary file.
func NewPTMySQLSummaryAction(id string, timeout time.Duration, cmd string, params *agentv1.StartActionRequest_PTMySQLSummaryParams) Action {
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
	cfg, err := a.buildMyCnfConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to build the config file: %w", err)
	}

	tmpFile, err := os.CreateTemp("", "pt-mysql-summary-action-*")
	if err != nil {
		return nil, fmt.Errorf("failed to create temporary file: %w", err)
	}
	defer os.Remove(tmpFile.Name()) //nolint:errcheck

	_, err = fmt.Fprint(tmpFile, cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to write to temporary file: %w", err)
	}
	tmpFile.Close() //nolint:errcheck

	cmd := exec.CommandContext(ctx, a.command, "--defaults-file="+tmpFile.Name()) //nolint:gosec
	cmd.Env = []string{fmt.Sprintf("PATH=%s", os.Getenv("PATH"))}
	cmd.Dir = "/"
	pdeathsig.Set(cmd, unix.SIGKILL)

	return cmd.CombinedOutput()
}

const myCnfTemplate = `[client]
{{if .Host}}host={{ .Host }}{{end}}
{{if .Port}}port={{ .Port }}{{end}}
{{if .User}}user={{ .User }}{{end}}
{{if .Password}}password={{ .Password }}{{end}}
{{if .Socket}}socket={{ .Socket }}{{end}}
`

// Creates a config file for MySQL client.
func (a *ptMySQLSummaryAction) buildMyCnfConfig() (string, error) {
	if a.params == nil {
		return "[client]\n", nil
	}

	if err := checkArgs(a.params.Host, a.params.Socket, a.params.Username, a.params.Password); err != nil {
		return "", fmt.Errorf("invalid parameters: %w", err)
	}

	tmpl, err := template.New("myCnf").Parse(myCnfTemplate)
	if err != nil {
		return "", fmt.Errorf("failed to parse the template: %w", err)
	}

	var myCnfBuffer bytes.Buffer
	myCnfParams := struct {
		User     string
		Password string
		Socket   string
		Host     string
		Port     uint32
	}{}

	if a.params.Socket != "" {
		myCnfParams.Socket = a.params.Socket
	} else {
		if a.params.Host != "" {
			myCnfParams.Host = a.params.Host
		}
		if a.params.Port > 0 && a.params.Port <= 65535 {
			myCnfParams.Port = a.params.Port
		}
	}

	if a.params.Username != "" {
		myCnfParams.User = a.params.Username
	}

	if a.params.Password != "" {
		myCnfParams.Password = a.params.Password
	}

	if err = tmpl.Execute(&myCnfBuffer, myCnfParams); err != nil {
		return "", fmt.Errorf("failed to execute myCnf template: %w", err)
	}

	return myCnfBuffer.String(), nil
}

func checkArgs(args ...string) error {
	for _, s := range args {
		if s == "" {
			continue
		}
		if strings.ContainsAny(s, "\n\r") {
			return ErrInvalidCharacter
		}
	}
	return nil
}

func (*ptMySQLSummaryAction) sealed() {}
