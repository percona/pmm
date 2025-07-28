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

// Package commands provides base commands and helpers.
package commands

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
	"text/template"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

var (
	// Ctx is a shared context for all requests.
	Ctx = context.Background()
	// CLICtx is context used for commands ran with context.
	CLICtx context.Context

	errExecutionNotImplemented = errors.New("execution is not supported")

	// SetupClientsEnabled defines if clients shall be setup during bootstrapping.
	SetupClientsEnabled = true
)

// Result is a common interface for all command results.
//
// In addition to methods of this interface, result is expected to work with json.Marshal.
type Result interface {
	Result()
	fmt.Stringer
}

// Command is a common interface for all commands.
//
// Command should:
//   - use logrus.Trace/Debug functions for debug logging;
//   - return result on success;
//   - return error on failure.
//
// Command should not:
//   - return both result and error;
//   - exit with logrus.Fatal, os.Exit, etc;
//   - use logrus.Print, logrus.Info and higher levels except:
//   - summary command (for progress output).
type Command interface {
	RunCmd() (Result, error)
}

// CommandWithContext is a new interface for commands.
//
// TODO remove Command above, rename CommandWithContext to Command.
type CommandWithContext interface {
	// TODO rename to Run
	RunWithContext(ctx context.Context) (Result, error)
}

// Credentials provides access to an external provider so that
// the username, password, or agent password can be managed
// externally, e.g. HashiCorp Vault, Ansible Vault, etc.
type Credentials struct {
	AgentPassword string `json:"agentpassword"`
	Password      string `json:"password"`
	Username      string `json:"username"`
}

// ReadFromSource parses a JSON file src and return
// a Credentials pointer containing the data.
func ReadFromSource(src string) (*Credentials, error) {
	creds := Credentials{"", "", ""}

	f, err := os.Lstat(src)
	if err != nil {
		return nil, fmt.Errorf("%w", err)
	}

	if f.Mode()&0o111 != 0 {
		return nil, fmt.Errorf("%w: %s", errExecutionNotImplemented, src)
	}

	// Read the file
	content, err := ReadFile(src)
	if err != nil {
		return nil, fmt.Errorf("%w", err)
	}

	if err := json.Unmarshal([]byte(content), &creds); err != nil {
		return nil, fmt.Errorf("%w", err)
	}

	return &creds, nil
}

// ErrorResponse defines the interface for error responses.
type ErrorResponse interface {
	error
	Code() int
}

// Error represents an error with additional information.
type Error struct {
	Code  int    `json:"code"`
	Error string `json:"error"`
}

// GetError converts an ErrorResponse to an Error.
func GetError(err ErrorResponse) Error {
	v := reflect.ValueOf(err)
	p := v.Elem().FieldByName("Payload")
	e := p.Elem().FieldByName("Message")
	return Error{
		Code:  err.Code(),
		Error: e.String(),
	}
}

// ParseTemplate parses the input text into a template.Template.
func ParseTemplate(text string) *template.Template {
	t := template.New("").Option("missingkey=error")
	return template.Must(t.Parse(strings.TrimSpace(text)))
}

// RenderTemplate renders given template with given data and returns result as string.
func RenderTemplate(t *template.Template, data interface{}) string {
	var buf bytes.Buffer
	if err := t.Execute(&buf, data); err != nil {
		logrus.Panicf("Failed to render response.\n%s.\nTemplate data: %#v.\nPlease report this bug.", err, data)
	}

	return strings.TrimSpace(buf.String()) + "\n"
}

var customLabelRE = regexp.MustCompile(`^([a-zA-Z_][a-zA-Z0-9_]*)=([^='", ]+)$`) //nolint:unused,varcheck

// ParseKeyValuePair parses values in key-value pair flags (e.g --custom-labels and --extra-dsn-params)
func ParseKeyValuePair(labels map[string]string) map[string]string {
	result := make(map[string]string)
	for k, v := range labels {
		v = strings.TrimSpace(v)
		if v == "" {
			continue
		}

		result[k] = v
	}
	return result
}

// ParseDisableCollectors parses --disable-collectors flag value.
func ParseDisableCollectors(collectors []string) []string {
	var disableCollectors []string

	if len(collectors) != 0 {
		for _, v := range collectors {
			disableCollector := strings.TrimSpace(v)
			if disableCollector == "" {
				continue
			}

			disableCollectors = append(disableCollectors, disableCollector)
		}
	}

	return disableCollectors
}

// ReadFile reads file from filepath if filepath is not empty.
func ReadFile(filePath string) (string, error) {
	if filePath == "" {
		return "", nil
	}

	content, err := os.ReadFile(filepath.Clean(filePath))
	if err != nil {
		return "", errors.Wrapf(err, "cannot load file in path %q", filePath)
	}

	return string(content), nil
}

// UsageTemplate is default kingping's usage template with tweaks:
// * FormatAllCommands is a copy of FormatCommands that ignores hidden flag;
// * subcommands are shown with FormatAllCommands.
var UsageTemplate = `{{define "FormatCommand"}}\
{{if .FlagSummary}} {{.FlagSummary}}{{end}}\
{{range .Args}} {{if not .Required}}[{{end}}<{{.Name}}>{{if .Value|IsCumulative}}...{{end}}{{if not .Required}}]{{end}}{{end}}\
{{end}}\

{{define "FormatCommands"}}\
{{range .FlattenedCommands}}\
{{if not .Hidden}}\
  {{.FullCommand}}{{if .Default}}*{{end}}{{template "FormatCommand" .}}
{{.Help|Wrap 4}}
{{end}}\
{{end}}\
{{end}}\

{{define "FormatAllCommands"}}\
{{range .FlattenedCommands}}\
{{if true}}\
  {{.FullCommand}}{{if .Default}}*{{end}}{{template "FormatCommand" .}}
{{.Help|Wrap 4}}
{{end}}\
{{end}}\
{{end}}\

{{define "FormatUsage"}}\
{{template "FormatCommand" .}}{{if .Commands}} <command> [<args> ...]{{end}}
{{if .Help}}
{{.Help|Wrap 0}}\
{{end}}\

{{end}}\

{{if .Context.SelectedCommand}}\
usage: {{.App.Name}} {{.Context.SelectedCommand}}{{template "FormatUsage" .Context.SelectedCommand}}
{{else}}\
usage: {{.App.Name}}{{template "FormatUsage" .App}}
{{end}}\
{{if .Context.Flags}}\
Flags:
{{.Context.Flags|FlagsToTwoColumns|FormatTwoColumns}}
{{end}}\
{{if .Context.Args}}\
Positional arguments:
{{.Context.Args|ArgsToTwoColumns|FormatTwoColumns}}
{{end}}\
{{if .Context.SelectedCommand}}\
{{if len .Context.SelectedCommand.Commands}}\
Subcommands:
{{template "FormatAllCommands" .Context.SelectedCommand}}
{{end}}\
{{else if .App.Commands}}\
Commands:
{{template "FormatCommands" .App}}
{{end}}\
`
