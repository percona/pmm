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

// Package cli stores cli configuration and common logic for commands.
package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"

	"github.com/alecthomas/kong"
	"github.com/sirupsen/logrus"

	"github.com/percona/pmm/admin/commands"
	"github.com/percona/pmm/admin/commands/inventory"
	"github.com/percona/pmm/admin/commands/management"
	"github.com/percona/pmm/admin/commands/pmm/client"
	"github.com/percona/pmm/admin/commands/pmm/server"
	"github.com/percona/pmm/admin/pkg/flags"
)

// GlobalFlagsGetter supports retrieving GlobalFlags.
type GlobalFlagsGetter interface {
	GetGlobalFlags() *flags.GlobalFlags
}

// Check interfaces.
var (
	_ GlobalFlagsGetter = &PMMAdminCommands{} //nolint:exhaustruct
	_ GlobalFlagsGetter = &PMMCommands{}      //nolint:exhaustruct
)

// PMMAdminCommands stores all commands, flags and arguments for the "pmm-admin" binary.
type PMMAdminCommands struct {
	flags.GlobalFlags

	Status     commands.StatusCommand       `cmd:"" help:"Show information about local pmm-agent"`
	Summary    commands.SummaryCommand      `cmd:"" help:"Fetch system data for diagnostics"`
	List       commands.ListCommand         `cmd:"" help:"Show Services and Agents running on this Node"`
	Config     commands.ConfigCommand       `cmd:"" help:"Configure local pmm-agent"`
	Annotate   commands.AnnotationCommand   `cmd:"" help:"Add an annotation to Grafana charts"`
	Unregister management.UnregisterCommand `cmd:"" help:"Unregister current Node from PMM Server"`
	Remove     management.RemoveCommand     `cmd:"" help:"Remove Service from monitoring"`
	Register   management.RegisterCommand   `cmd:"" help:"Register current Node with PMM Server"`
	Add        management.AddCommand        `cmd:"" help:"Add Service to monitoring"`
	Inventory  inventory.InventoryCommand   `cmd:"" hidden:"" help:"Inventory commands"`
	Version    commands.VersionCommand      `cmd:"" help:"Print version"`
	Completion commands.CompletionCommand   `cmd:"" help:"Outputs shell code for initialising tab completions"`
}

// Run function is a top-level function which handles running all commands
// in a standard way based on the interface they implement.
func (c *PMMAdminCommands) Run(ctx *kong.Context, globals *flags.GlobalFlags) error {
	return run(ctx, globals)
}

// GetGlobalFlags returns the global flags for PMMAdminCommands.
func (c *PMMAdminCommands) GetGlobalFlags() *flags.GlobalFlags {
	return &c.GlobalFlags
}

// PMMCommands stores all commands, flags and arguments for the "pmm" binary.
type PMMCommands struct {
	flags.GlobalFlags

	Server     server.BaseCommand         `cmd:"" help:"PMM server related commands"`
	Client     client.BaseCommand         `cmd:"" help:"PMM client related commands"`
	Completion commands.CompletionCommand `cmd:"" help:"Outputs shell code for initialising tab completions"`
}

// GetGlobalFlags returns the global flags for PMMAdminCommands.
func (c *PMMCommands) GetGlobalFlags() *flags.GlobalFlags {
	return &c.GlobalFlags
}

// Run function is a top-level function which handles running all commands
// in a standard way based on the interface they implement.
func (c *PMMCommands) Run(ctx *kong.Context, globals *flags.GlobalFlags) error {
	return run(ctx, globals)
}

// CmdRunner represents a command to be run without arguments.
type CmdRunner interface {
	RunCmd() (commands.Result, error)
}

// CmdGlobalFlagsRunner represents a command to be run with global CLI flags.
type CmdGlobalFlagsRunner interface {
	RunCmd(*flags.GlobalFlags) (commands.Result, error)
}

// CmdWithContextRunner represents a command to be run with context.
type CmdWithContextRunner interface {
	RunCmdWithContext(context.Context, *flags.GlobalFlags) (commands.Result, error)
}

func run(ctx *kong.Context, globals *flags.GlobalFlags) error {
	var res commands.Result
	var err error

	i := ctx.Selected().Target.Addr().Interface()

	switch cmd := i.(type) {
	case CmdWithContextRunner:
		res, err = cmd.RunCmdWithContext(commands.CLICtx, globals)
	case CmdGlobalFlagsRunner:
		res, err = cmd.RunCmd(globals)
	case CmdRunner:
		res, err = cmd.RunCmd()
	default:
		panic("The command does not implement RunCmd()")
	}

	return printResponse(globals, res, err)
}

func printResponse(opts *flags.GlobalFlags, res commands.Result, err error) error {
	logrus.Debugf("Result: %#v", res)
	logrus.Debugf("Error: %#v", err)

	switch err := err.(type) { //nolint:errorlint
	case nil:
		printSuccessResult(opts, res)
		os.Exit(0)

	case commands.ErrorResponse:
		printErrorResponse(opts, err)
		os.Exit(1)

	case *exec.ExitError: // from config command that execs `pmm-agent setup`
		if res != nil {
			printExitError(opts, res, err)
			os.Exit(err.ExitCode())
		}
	}

	return err
}

func printSuccessResult(opts *flags.GlobalFlags, res commands.Result) {
	if opts.JSON {
		b, jErr := json.Marshal(res)
		if jErr != nil {
			logrus.Infof("Result: %#v.", res)
			logrus.Panicf("Failed to marshal result to JSON.\n%s.\nPlease report this bug.", jErr)
		}
		fmt.Printf("%s\n", b) //nolint:forbidigo
	} else {
		fmt.Println(res.String()) //nolint:forbidigo
	}
}

func printErrorResponse(opts *flags.GlobalFlags, err commands.ErrorResponse) {
	e := commands.GetError(err)

	if opts.JSON {
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
}

func printExitError(opts *flags.GlobalFlags, res commands.Result, err *exec.ExitError) {
	if opts.JSON {
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
}
