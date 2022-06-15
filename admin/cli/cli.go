// pmm-admin
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

package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"reflect"

	"github.com/alecthomas/kong"
	"github.com/sirupsen/logrus"

	"github.com/percona/pmm/admin/cli/opts"
	"github.com/percona/pmm/admin/commands"
	"github.com/percona/pmm/admin/commands/inventory"
	"github.com/percona/pmm/admin/commands/management"
	"github.com/percona/pmm/version"
)

var (
	isJSON = false
	CLI    = opts.Opts{
		SetupClients: true,
	}
)

type CLIGlobalFlags struct {
	ServerURL          string      `name:"server-url" placeholder:"SERVER-URL" help:"PMM Server URL in https://username:password@pmm-server-host/ format"`
	ServerInsecureTls  bool        `name:"server-insecure-tls" help:"Skip PMM Server TLS certificate validation"`
	Debug              bool        `name:"debug" help:"Enable debug logging"`
	Trace              bool        `name:"trace" help:"Enable trace logging (implies debug)"`
	PMMAgentListenPort uint32      `name:"pmm-agent-listen-port" default:"${defaultListenPort}" help:"Set listen port of pmm-agent"`
	JSON               jsonFlag    `name:"json" help:"Enable JSON output"`
	Version            versionFlag `name:"version" short:"v" help:"Show application version"`
}

type versionFlag bool

func (v versionFlag) BeforeApply(app *kong.Kong, ctx *kong.Context) error {
	if isJSON {
		fmt.Println(version.FullInfoJSON())
	} else {
		fmt.Println(version.FullInfo())
	}
	os.Exit(0)

	return nil
}

type jsonFlag bool

func (v jsonFlag) BeforeApply() error {
	isJSON = true
	return nil
}

type CLIFlags struct {
	CLIGlobalFlags

	Status     commands.StatusCmd       `cmd:"" help:"Show information about local pmm-agent"`
	Summary    commands.SummaryCmd      `cmd:"" help:"Fetch system data for diagnostics"`
	List       commands.ListCmd         `cmd:"" help:"Show Services and Agents running on this Node"`
	Config     commands.ConfigCmd       `cmd:"" help:"Configure local pmm-agent"`
	Annotate   commands.AnnotateCmd     `cmd:"" help:"Add an annotation to Grafana charts"`
	Unregister management.UnregisterCmd `cmd:"" help:"Unregister current Node from PMM Server"`
	Remove     management.RemoveCmd     `cmd:"" help:"Remove Service from monitoring"`
	Register   management.RegisterCmd   `cmd:"" help:"Register current Node with PMM Server"`
	Add        management.AddCmd        `cmd:"" help:"Add Service to monitoring"`
	Inventory  inventory.InventoryCmd   `cmd:"" hidden:"" help:"Inventory commands"`
}

func (c *CLIFlags) Run(ctx *kong.Context) error {
	in := []reflect.Value{}
	method := getMethod(ctx.Selected().Target, "RunCmdWithContext")
	if method.IsValid() {
		in = append(in, reflect.ValueOf(CLI.Ctx))
	} else {
		method = getMethod(ctx.Selected().Target, "RunCmd")
	}

	out := method.Call(in)
	if !out[1].IsNil() {
		return out[1].Interface().(error)
	}

	return PrintResponse(&c.CLIGlobalFlags, out[0].Interface().(commands.Result), nil)
}

func PrintResponse(opts *CLIGlobalFlags, res commands.Result, err error) error {
	logrus.Debugf("Result: %#v", res)
	logrus.Debugf("Error: %#v", err)

	switch err := err.(type) {
	case nil:
		if (*opts).JSON {
			b, jErr := json.Marshal(res)
			if jErr != nil {
				logrus.Infof("Result: %#v.", res)
				logrus.Panicf("Failed to marshal result to JSON.\n%s.\nPlease report this bug.", jErr)
			}
			fmt.Printf("%s\n", b) //nolint:forbidigo
		} else {
			fmt.Println(res.String()) //nolint:forbidigo
		}

		os.Exit(0)

	case commands.ErrorResponse:
		e := commands.GetError(err)

		if (*opts).JSON {
			b, jErr := json.Marshal(e)
			if jErr != nil {
				logrus.Infof("Error response: %#v.", e)
				logrus.Panicf("Failed to marshal error response to JSON.\n%s.\nPlease report this bug.", jErr)
			}
			fmt.Printf("%s\n", b) //nolint:forbidigo
		} else {
			msg := e.Error
			if e.Code == 401 {
				msg += ". Please check username and password."
			}
			fmt.Println(msg) //nolint:forbidigo
		}

		os.Exit(1)

	case *exec.ExitError: // from config command that execs `pmm-agent setup`
		if (*opts).JSON {
			b, jErr := json.Marshal(res)
			if jErr != nil {
				logrus.Infof("Result: %#v.", res)
				logrus.Panicf("Failed to marshal result to JSON.\n%s.\nPlease report this bug.", jErr)
			}
			fmt.Printf("%s\n", b) //nolint:forbidigo
		} else {
			fmt.Println(res.String()) //nolint:forbidigo
		}

		if err.Stderr != nil {
			logrus.Debugf("%s, stderr:\n%s", err.String(), err.Stderr)
		}

		os.Exit(err.ExitCode())
	}

	return err
}

func getMethod(value reflect.Value, name string) reflect.Value {
	method := value.MethodByName(name)
	if !method.IsValid() {
		if value.CanAddr() {
			method = value.Addr().MethodByName(name)
		}
	}
	return method
}
