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

package docker

import (
	"context"
	"io"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"

	"github.com/percona/pmm/admin/pkg/docker"
)

//go:generate ../../../../../bin/mockery -name=DockerFunctions -case=snake -inpkg -testonly

type DockerFunctions interface {
	ChangeServerPassword(ctx context.Context, containerID, newPassword string) error
	CreateVolume(ctx context.Context, volumeName string) (*types.Volume, error)
	FindServerContainers(ctx context.Context) ([]types.Container, error)
	GetDockerClient() docker.DockerClient
	HaveDockerAccess(ctx context.Context) bool
	InstallDocker() error
	IsDockerInstalled() (bool, error)
	PullImage(ctx context.Context, dockerImage string, opts types.ImagePullOptions) (io.Reader, error)
	RunContainer(ctx context.Context, config *container.Config, hostConfig *container.HostConfig, containerName string) (string, error)
	WaitForHealthyContainer(ctx context.Context, containerID string) <-chan docker.WaitHealthyResponse
}
