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

	"github.com/sirupsen/logrus"

	"github.com/percona/pmm/admin/cli/flags"
	"github.com/percona/pmm/admin/commands"
	"github.com/percona/pmm/admin/pkg/docker"
	"github.com/percona/pmm/admin/services/api_server"
)

// RunCommand is used by Kong for CLI flags and commands.
type RunCommand struct {
	DockerImage       string `default:"percona/pmm-server:2" help:"Docker image to use for updating PMM to the latest version"`
	DisableSelfUpdate bool   `help:"Disables self-update of pmm-server-upgrade"`

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
	server.Run(ctx)

	return &runResult{}, nil
}
