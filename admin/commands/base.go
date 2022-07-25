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

// Package commands provides base commands and helpers.
package commands

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
	"text/template"

	"github.com/go-openapi/runtime"
	httptransport "github.com/go-openapi/runtime/client"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"github.com/percona/pmm/admin/agentlocal"
	inventorypb "github.com/percona/pmm/api/inventorypb/json/client"
	managementpb "github.com/percona/pmm/api/managementpb/json/client"
	serverpb "github.com/percona/pmm/api/serverpb/json/client"
	"github.com/percona/pmm/utils/tlsconfig"
)

var (
	// Ctx is a shared context for all requests.
	Ctx = context.Background()

	errExecutionNotImplemented = errors.New("execution is not supported")
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
//  * use logrus.Trace/Debug functions for debug logging;
//  * return result on success;
//  * return error on failure.
//
// Command should not:
//  * return both result and error;
//  * exit with logrus.Fatal, os.Exit, etc;
//  * use logrus.Print, logrus.Info and higher levels except:
//    * summary command (for progress output).
type Command interface {
	Run() (Result, error)
}

// TODO remove Command above, rename CommandWithContext to Command
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

type ErrorResponse interface {
	error
	Code() int
}

type Error struct {
	Code  int    `json:"code"`
	Error string `json:"error"`
}

func GetError(err ErrorResponse) Error {
	v := reflect.ValueOf(err)
	p := v.Elem().FieldByName("Payload")
	e := p.Elem().FieldByName("Message")
	return Error{
		Code:  err.Code(),
		Error: e.String(),
	}
}

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

type globalFlagsValues struct {
	ServerURL          *url.URL
	ServerInsecureTLS  bool
	Debug              bool
	Trace              bool
	PMMAgentListenPort uint32
}

// GlobalFlags contains pmm-admin core flags values.
var GlobalFlags globalFlagsValues

var customLabelRE = regexp.MustCompile(`^([a-zA-Z_][a-zA-Z0-9_]*)=([^='", ]+)$`)

// ParseCustomLabels parses --custom-labels flag value.
//
// Note that quotes around value are parsed and removed by shell before this function is called.
// E.g. the value of [[--custom-labels='region=us-east1, mylabel=mylab-22']] will be received by this function
// as [[region=us-east1, mylabel=mylab-22]].
func ParseCustomLabels(labels string) (map[string]string, error) {
	result := make(map[string]string)
	parts := strings.Split(labels, ",")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		submatches := customLabelRE.FindStringSubmatch(part)
		if submatches == nil {
			return nil, errors.New("wrong custom label format")
		}
		result[submatches[1]] = submatches[2]
	}
	return result, nil
}

// ParseDisableCollectors parses --disable-collectors flag value.
func ParseDisableCollectors(collectors string) []string {
	var disableCollectors []string

	if collectors != "" {
		for _, v := range strings.Split(collectors, ",") {
			disableCollector := strings.TrimSpace(v)
			if disableCollector != "" {
				disableCollectors = append(disableCollectors, disableCollector)
			}
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

type nginxError string

func (e nginxError) Error() string {
	return "response from nginx: " + string(e)
}

func (e nginxError) GoString() string {
	return fmt.Sprintf("nginxError(%q)", string(e))
}

// SetupClients configures local and PMM Server API clients.
func SetupClients(ctx context.Context, serverURL string) {
	if serverURL == "" {
		status, err := agentlocal.GetStatus(agentlocal.DoNotRequestNetworkInfo)
		if err != nil {
			if err == agentlocal.ErrNotSetUp { //nolint:errorlint,goerr113
				logrus.Fatalf("Failed to get PMM Server parameters from local pmm-agent: %s.\n"+
					"Please run `pmm-admin config` with --server-url flag.", err)
			}

			if err == agentlocal.ErrNotConnected { //nolint:errorlint,goerr113
				logrus.Fatalf("Failed to get PMM Server parameters from local pmm-agent: %s.\n", err)
			}
			logrus.Fatalf("Failed to get PMM Server parameters from local pmm-agent: %s.\n"+
				"Please use --server-url flag to specify PMM Server URL.", err)
		}
		GlobalFlags.ServerURL, _ = url.Parse(status.ServerURL)
		GlobalFlags.ServerInsecureTLS = status.ServerInsecureTLS
	} else {
		var err error
		GlobalFlags.ServerURL, err = url.Parse(serverURL)
		if err != nil {
			logrus.Fatalf("Invalid PMM Server URL %q: %s.", serverURL, err)
		}
		if GlobalFlags.ServerURL.Path == "" {
			GlobalFlags.ServerURL.Path = "/"
		}
		switch GlobalFlags.ServerURL.Scheme {
		case "http", "https":
			// nothing
		default:
			logrus.Fatalf("Invalid PMM Server URL %q: scheme (https:// or http://) is missing.", serverURL)
		}
		if GlobalFlags.ServerURL.Host == "" {
			logrus.Fatalf("Invalid PMM Server URL %q: host is missing.", serverURL)
		}
	}

	// use JSON APIs over HTTP/1.1
	transport := httptransport.New(GlobalFlags.ServerURL.Host, GlobalFlags.ServerURL.Path, []string{GlobalFlags.ServerURL.Scheme})
	if u := GlobalFlags.ServerURL.User; u != nil {
		password, _ := u.Password()
		transport.DefaultAuthentication = httptransport.BasicAuth(u.Username(), password)
	}
	transport.SetLogger(logrus.WithField("component", "server-transport"))
	transport.SetDebug(GlobalFlags.Debug || GlobalFlags.Trace)
	transport.Context = ctx

	// set error handlers for nginx responses if pmm-managed is down
	errorConsumer := runtime.ConsumerFunc(func(reader io.Reader, data interface{}) error {
		b, _ := io.ReadAll(reader)
		return nginxError(string(b))
	})
	transport.Consumers = map[string]runtime.Consumer{
		runtime.JSONMime:    runtime.JSONConsumer(),
		"application/zip":   runtime.ByteStreamConsumer(),
		runtime.HTMLMime:    errorConsumer,
		runtime.TextMime:    errorConsumer,
		runtime.DefaultMime: errorConsumer,
	}

	// disable HTTP/2, set TLS config
	httpTransport := transport.Transport.(*http.Transport)
	httpTransport.TLSNextProto = make(map[string]func(string, *tls.Conn) http.RoundTripper)
	if GlobalFlags.ServerURL.Scheme == "https" {
		httpTransport.TLSClientConfig = tlsconfig.Get()
		httpTransport.TLSClientConfig.ServerName = GlobalFlags.ServerURL.Hostname()
		httpTransport.TLSClientConfig.InsecureSkipVerify = GlobalFlags.ServerInsecureTLS
	}

	inventorypb.Default.SetTransport(transport)
	managementpb.Default.SetTransport(transport)
	serverpb.Default.SetTransport(transport)
}

// check interfaces
var (
	_ error          = nginxError("")
	_ fmt.GoStringer = nginxError("")
)

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
