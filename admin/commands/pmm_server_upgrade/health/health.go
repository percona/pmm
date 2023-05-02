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

// Package health reports if pmm-server-upgrade is healthy.
// The package is used in distroless Docker images to report health
// instead of bundling other utils to check health.
package health

import (
	"context"
	"io/fs"
	"os"

	"github.com/pkg/errors"

	"github.com/percona/pmm/admin/cli/flags"
	"github.com/percona/pmm/admin/commands"
	"github.com/percona/pmm/admin/pkg/apiserver"
)

// HealthCommand is used by Kong for CLI flags and commands.
type HealthCommand struct{}

type healthResult struct{}

// Result is a command run result.
func (r *healthResult) Result() {}

// String stringifies command result.
func (r *healthResult) String() string {
	return "ok"
}

// BeforeApply is run before the command is applied.
func (c *HealthCommand) BeforeApply() error {
	commands.SetupClientsEnabled = false
	return nil
}

// RunCmdWithContext runs command
func (c *HealthCommand) RunCmdWithContext(ctx context.Context, globals *flags.GlobalFlags) (commands.Result, error) {
	s, err := os.Stat(apiserver.SocketPath)
	if err != nil {
		return nil, err
	}

	if s.Mode()&fs.ModeSocket != fs.ModeSocket {
		return nil, errors.New("File is not a socket file")
	}

	return &healthResult{}, nil
}
