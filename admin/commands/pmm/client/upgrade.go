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

package client

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/sirupsen/logrus"

	"github.com/percona/pmm/admin/cli/flags"
	"github.com/percona/pmm/admin/commands"
	"github.com/percona/pmm/admin/pkg/common"
)

// UpgradeCommand is used by Kong for CLI flags and commands.
type UpgradeCommand struct {
	Version string `name:"use-version" help:"PMM Client version to upgrade to (default: latest)"`
}

type upgradeResult struct{}

// Result is a command run result.
func (res *upgradeResult) Result() {}

// String stringifies command result.
func (res *upgradeResult) String() string {
	return "ok"
}

// RunCmdWithContext runs install command.
func (c *UpgradeCommand) RunCmdWithContext(ctx context.Context, _ *flags.GlobalFlags) (commands.Result, error) {
	distributionType := common.DetectDistributionType()

	var err error
	switch distributionType {
	case common.PackageManager:
		err = c.upgradeViaPackageManager(ctx)
	default:
		logrus.Panicf("Not supported distribution type %d", distributionType)
	}

	if err != nil {
		return nil, err
	}

	return &upgradeResult{}, nil
}

func (c *UpgradeCommand) upgradeViaPackageManager(ctx context.Context) error {
	cmds, err := c.getUpgradeCommands()
	if err != nil {
		return err
	}

	for _, cmd := range cmds {
		logrus.Infof("Running command %q", strings.Join(cmd, " "))

		cmd := exec.CommandContext(ctx, cmd[0], cmd[1:]...) //nolint:gosec
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		if err := cmd.Run(); err != nil {
			return err
		}
	}

	return nil
}

// ErrNoUpgradeCommandFound is returned when yum/dnf/apt/... Package manager cannot be detected.
var ErrNoUpgradeCommandFound = fmt.Errorf("NoUpgradeCommandFound")

func (c *UpgradeCommand) getUpgradeCommands() ([][]string, error) {
	cmd, err := common.LookupCommand("dnf")
	if err != nil {
		return nil, err
	}

	if cmd != "" {
		return [][]string{{"dnf", "upgrade", "-y", fmt.Sprintf("pmm2-client%s", c.getVersionSuffix(dnf))}}, nil
	}

	cmd, err = common.LookupCommand("yum")
	if err != nil {
		return nil, err
	}

	if cmd != "" {
		return [][]string{{"yum", "upgrade", "-y", fmt.Sprintf("pmm2-client%s", c.getVersionSuffix(yum))}}, nil
	}

	cmd, err = common.LookupCommand("apt")
	if err != nil {
		return nil, err
	}

	if cmd != "" {
		return [][]string{
			{"percona-release", "enable", "pmm2-client", "release"},
			{"apt", "update"},
			{"apt", "install", "--only-upgrade", "-y", fmt.Sprintf("pmm2-client%s", c.getVersionSuffix(apt))},
		}, nil
	}

	return nil, fmt.Errorf("%w: cannot detect package manager (yum/dnf/apt)", ErrNoUpgradeCommandFound)
}

const versionLatest = "latest"

type packageManager int

const (
	dnf packageManager = iota
	yum
	apt
)

func (c *UpgradeCommand) getVersionSuffix(pm packageManager) string {
	if c.Version == "" || c.Version == versionLatest {
		return ""
	}

	switch pm {
	case dnf:
		return fmt.Sprintf("-%s", c.Version)
	case yum:
		return fmt.Sprintf("-%s", c.Version)
	case apt:
		return fmt.Sprintf("=%s", c.Version)
	}

	logrus.Panic("Invalid package manager provided")
	return ""
}
