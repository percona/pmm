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
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/sirupsen/logrus"

	"github.com/percona/pmm/admin/commands"
	"github.com/percona/pmm/admin/pkg/client"
	"github.com/percona/pmm/admin/pkg/client/tarball"
	"github.com/percona/pmm/admin/pkg/common"
	"github.com/percona/pmm/admin/pkg/flags"
)

// UpgradeCommand is used by Kong for CLI flags and commands.
type UpgradeCommand struct {
	Distribution string `enum:"autodetect,package-manager,tarball,docker" default:"autodetect" help:"Type of PMM Client distribution. One of: [${enum}]. Default: ${default}"` //nolint:lll
	Version      string `name:"use-version" help:"PMM Client version to upgrade to (default: latest)"`

	InstallPath  string `group:"Tarball flags" default:"/usr/local/percona/pmm" help:"Path where PMM Client is installed"`
	User         string `group:"Tarball flags" help:"Set file ownership instead of the current user"`
	Group        string `group:"Tarball flags" help:"Set group ownership instead of the current group"`
	SkipChecksum bool   `group:"Tarball flags" help:"Skip checksum validation of the downloaded files"`
}

type distributionType string

const (
	distributionAutodetect     distributionType = "autodetect"
	distributionPackageManager distributionType = "package-manager"
	distributionTar            distributionType = "tarball"
	distributionDocker         distributionType = "docker"
)

const versionLatest = "latest"

type packageManager int

const (
	dnf packageManager = iota
	yum
	apt
)

type upgradeResult struct{}

// Result is a command run result.
func (res *upgradeResult) Result() {}

// String stringifies command result.
func (res *upgradeResult) String() string {
	return "ok"
}

// RunCmdWithContext runs install command.
func (c *UpgradeCommand) RunCmdWithContext(ctx context.Context, _ *flags.GlobalFlags) (commands.Result, error) {
	distributionType, err := c.distributionType(ctx)
	if err != nil {
		return nil, err
	}

	switch distributionType {
	case client.PackageManager:
		err = c.upgradeViaPackageManager(ctx)
	case client.Tarball:
		err = c.upgradeViaTarball(ctx)
	default:
		logrus.Panicf("Unsupported distribution type %q", distributionType)
	}

	if err != nil {
		return nil, err
	}

	return &upgradeResult{}, nil
}

func (c *UpgradeCommand) distributionType(ctx context.Context) (client.DistributionType, error) {
	var distType client.DistributionType
	var err error
	switch distributionType(c.Distribution) {
	case distributionAutodetect:
		distType, err = client.DetectDistributionType(ctx, c.InstallPath)
		if err != nil {
			return client.Unknown, err
		}
	case distributionPackageManager:
		distType = client.PackageManager
	case distributionTar:
		distType = client.Tarball
	case distributionDocker:
		distType = client.Docker
	}

	return distType, nil
}

func (c *UpgradeCommand) upgradeViaTarball(ctx context.Context) error {
	t := tarball.Base{
		InstallPath:  c.InstallPath,
		User:         c.User,
		Group:        c.Group,
		Version:      c.Version,
		SkipChecksum: c.SkipChecksum,
		IsUpgrade:    true,
	}
	if err := t.Install(ctx); err != nil {
		return err
	}

	return nil
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
	cmd, err := common.DetectPackageManager()
	if err != nil {
		return nil, err
	}

	switch cmd {
	case common.Dnf:
		return [][]string{
			{"dnf", "upgrade", "-y", fmt.Sprintf("pmm-client%s", c.getVersionSuffix(dnf))},
		}, nil
	case common.Yum:
		return [][]string{
			{"yum", "upgrade", "-y", fmt.Sprintf("pmm-client%s", c.getVersionSuffix(yum))},
		}, nil
	case common.Apt:
		return [][]string{
			{"percona-release", "enable", "pmm-client", "release"},
			{"apt", "update"},
			{"apt", "install", "--only-upgrade", "-y", fmt.Sprintf("pmm-client%s", c.getVersionSuffix(apt))},
		}, nil
	case common.UnknownPackageManager:
		return nil, fmt.Errorf("%w: cannot detect package manager (yum/dnf/apt)", ErrNoUpgradeCommandFound)
	}

	return nil, fmt.Errorf("%w: cannot detect package manager (yum/dnf/apt)", ErrNoUpgradeCommandFound)
}

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
		return fmt.Sprintf("=%s*", c.Version)
	}

	logrus.Panic("Invalid package manager provided")
	return ""
}
