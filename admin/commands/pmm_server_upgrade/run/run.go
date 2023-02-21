// Copyright 2023 Percona LLC
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

// Package run holds logic for running pmm-server-upgrade.
package run

import (
	"context"
	"fmt"
	"path/filepath"
	"runtime"

	"github.com/sirupsen/logrus"

	"github.com/percona/pmm/admin/cli/flags"
	"github.com/percona/pmm/admin/commands"
	"github.com/percona/pmm/admin/pkg/api_server"
	"github.com/percona/pmm/admin/pkg/docker"
	"github.com/percona/pmm/admin/pkg/self_update"
)

// RunCommand is used by Kong for CLI flags and commands.
type RunCommand struct {
	DockerImage string `group:"PMM Server upgrade" default:"percona/pmm-server:2" help:"Docker image to use for updating PMM to the latest version"`

	DisableSelfUpdate          bool   `group:"Self update" help:"Disables self-update of pmm-server-upgrade"`
	SelfUpdateDockerImage      string `group:"Self update" default:"percona/pmm-server-upgrade:2" help:"Docker image to use for self-updating pmm-server-upgrade"`
	SelfUpdateDisableImagePull bool   `group:"Self update" help:"Disables pulling a new docker image of pmm-server-upgrade before self-update"`
	SelfUpdateTriggerOnStart   bool   `group:"Self update" help:"Trigger self-update check on start" `

	docker  containerManager
	globals *flags.GlobalFlags
}

type runResult struct{}

// Result is a command run result.
func (r *runResult) Result() {}

// String stringifies command result.
func (r *runResult) String() string {
	return "Exiting"
}

// BeforeApply is run before the command is applied.
func (c *RunCommand) BeforeApply() error {
	commands.SetupClientsEnabled = false
	return nil
}

// RunCmdWithContext runs command
func (c *RunCommand) RunCmdWithContext(ctx context.Context, globals *flags.GlobalFlags) (commands.Result, error) {
	c.configureLogger()

	logrus.Info("Starting PMM Server Upgrade")

	c.globals = globals

	if c.docker == nil {
		d, err := docker.New(nil)
		if err != nil {
			return nil, err
		}

		c.docker = d
	}

	if !c.docker.HaveDockerAccess(ctx) {
		return nil, fmt.Errorf("cannot access Docker. Make sure this container has access to the Docker socket")
	}

	// API server
	server := api_server.New(c.DockerImage)
	server.EnableDebug = c.globals.EnableDebug
	updateService := server.Start(ctx)

	// Self update
	if !c.DisableSelfUpdate {
		updater := self_update.New(
			c.docker,
			c.SelfUpdateDockerImage,
			c.SelfUpdateDisableImagePull,
			server,
			updateService,
			c.SelfUpdateTriggerOnStart,
		)
		updater.Start(ctx)
	}

	<-ctx.Done()

	return &runResult{}, nil
}

func (c *RunCommand) configureLogger() {
	// Set custom logrus formatter for this command
	logrus.SetFormatter(&logrus.TextFormatter{
		// Enable multiline-friendly formatter in both development (with terminal) and production (without terminal):
		// https://github.com/sirupsen/logrus/blob/839c75faf7f98a33d445d181f3018b5c3409a45e/text_formatter.go#L176-L178
		ForceColors:     true,
		FullTimestamp:   true,
		TimestampFormat: "2006-01-02T15:04:05.000-07:00",

		CallerPrettyfier: func(f *runtime.Frame) (function string, file string) {
			_, function = filepath.Split(f.Function)

			// keep a single directory name as a compromise between brevity and unambiguity
			var dir string
			dir, file = filepath.Split(f.File)
			dir = filepath.Base(dir)
			file = fmt.Sprintf("%s/%s:%d", dir, file, f.Line)

			return
		},
	})
}
