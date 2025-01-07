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

package client

import (
	"context"

	"github.com/percona/pmm/admin/commands"
	"github.com/percona/pmm/admin/pkg/client/tarball"
	"github.com/percona/pmm/admin/pkg/flags"
)

// InstallCommand is used by Kong for CLI flags and commands.
type InstallCommand struct {
	InstallPath  string `default:"/usr/local/percona/pmm" help:"Path where PMM Client shall be installed"`
	User         string `help:"Set file ownership instead of the current user"`
	Group        string `help:"Set group ownership instead of the current group"`
	Version      string `name:"use-version" help:"PMM Server version to install (default: latest)"`
	SkipChecksum bool   `help:"Skip checksum validation of the downloaded files"`
}

type installResult struct{}

// Result is a command run result.
func (res *installResult) Result() {}

// String stringifies command result.
func (res *installResult) String() string {
	return "ok"
}

// RunCmdWithContext runs install command.
func (c *InstallCommand) RunCmdWithContext(ctx context.Context, _ *flags.GlobalFlags) (commands.Result, error) {
	t := tarball.Base{
		InstallPath:  c.InstallPath,
		User:         c.User,
		Group:        c.Group,
		Version:      c.Version,
		SkipChecksum: c.SkipChecksum,
		IsUpgrade:    false,
	}
	if err := t.Install(ctx); err != nil {
		return nil, err
	}

	return &installResult{}, nil
}
