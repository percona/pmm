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

// Package docker holds the "pmm server install docker" command.
package docker

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"

	"github.com/percona/pmm/admin/pkg/docker"
)

// BaseCommand contains all commands for docker.
type BaseCommand struct {
	Install InstallCommand `cmd:"" help:"Install PMM server"`
	Upgrade UpgradeCommand `cmd:"" help:"Upgrade PMM server"`
}

type prepareOpts struct {
	// Shall Docker be installed, if not available?
	install bool
}

func prepareDocker(ctx context.Context, dockerFn Functions, opts prepareOpts) (Functions, error) { //nolint:ireturn
	if dockerFn == nil {
		d, err := docker.New(nil)
		if err != nil {
			return nil, err
		}

		dockerFn = d
	}

	if opts.install {
		if err := installDocker(ctx, dockerFn); err != nil {
			return nil, err
		}
	}

	if !dockerFn.HaveDockerAccess(ctx) {
		return nil, fmt.Errorf("%w: docker is either not running or this user has no access to Docker. Try running as root", ErrDockerNoAccess)
	}

	return dockerFn, nil
}

func installDocker(ctx context.Context, dockerFn Functions) error {
	isInstalled, err := dockerFn.IsDockerInstalled()
	if err != nil {
		return err
	}

	if isInstalled {
		return nil
	}

	logrus.Infoln("Installing Docker")
	if err = dockerFn.InstallDocker(ctx); err != nil { //nolint:revive
		return err
	}

	return nil
}
