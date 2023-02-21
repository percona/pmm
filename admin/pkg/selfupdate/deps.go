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

package selfupdate

import (
	"context"
	"io"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"

	"github.com/percona/pmm/admin/pkg/docker"
	"github.com/percona/pmm/admin/services/update"
)

type containerManager interface {
	imageManager

	ContainerInspect(ctx context.Context, containerID string) (types.ContainerJSON, error)
	ContainerStop(ctx context.Context, containerID string, timeout *time.Duration) error
	ContainerUpdate(ctx context.Context, containerID string, updateConfig container.UpdateConfig) (container.ContainerUpdateOKBody, error)
	RunContainer(ctx context.Context, config *container.Config, hostConfig *container.HostConfig, containerName string) (string, error)
	WaitForHealthyContainer(ctx context.Context, containerID string) <-chan docker.WaitHealthyResponse
}

type imageManager interface {
	ImageInspectWithRaw(ctx context.Context, imageID string) (types.ImageInspect, []byte, error)
	PullImage(ctx context.Context, dockerImage string, opts types.ImagePullOptions) (io.Reader, error)
}

type updateRunningChecker interface {
	IsAnyUpdateRunning() bool
}

type serverStartStopper interface {
	Start(ctx context.Context) *update.Server
	Stop()
}
