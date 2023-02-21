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

package docker

import (
	"context"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/sirupsen/logrus"
)

// WaitHealthyResponse holds information about container being healthy.
type WaitHealthyResponse struct {
	Healthy bool
	Error   error
}

// RunContainer creates and runs a container. It returns the container ID.
func (b *Base) RunContainer(ctx context.Context, config *container.Config, hostConfig *container.HostConfig, containerName string) (string, error) {
	res, err := b.Cli.ContainerCreate(ctx, config, hostConfig, nil, nil, containerName)
	if err != nil {
		return "", err
	}

	if err := b.Cli.ContainerStart(ctx, res.ID, types.ContainerStartOptions{}); err != nil { //nolint:exhaustruct
		return "", err
	}

	return res.ID, nil
}

// ContainerInspect returns information about a container.
func (b *Base) ContainerInspect(ctx context.Context, containerID string) (types.ContainerJSON, error) {
	return b.Cli.ContainerInspect(ctx, containerID)
}

// ContainerStop stops a container.
func (b *Base) ContainerStop(ctx context.Context, containerID string, timeout *time.Duration) error {
	return b.Cli.ContainerStop(ctx, containerID, timeout)
}

// ContainerUpdate updates container configuration.
func (b *Base) ContainerUpdate(ctx context.Context, containerID string, updateConfig container.UpdateConfig) (container.ContainerUpdateOKBody, error) {
	return b.Cli.ContainerUpdate(ctx, containerID, updateConfig)
}

// ContainerWait waits until a container is in a specific state.
func (b *Base) ContainerWait(ctx context.Context, containerID string, condition container.WaitCondition) (<-chan container.ContainerWaitOKBody, <-chan error) {
	return b.Cli.ContainerWait(ctx, containerID, condition)
}

// ContainerList lists containers according to filters
func (b *Base) ContainerList(ctx context.Context, options types.ContainerListOptions) ([]types.Container, error) {
	return b.Cli.ContainerList(ctx, options)
}

// FindServerContainers finds all containers running PMM Server.
func (b *Base) FindServerContainers(ctx context.Context) ([]types.Container, error) {
	return b.Cli.ContainerList(ctx, types.ContainerListOptions{ //nolint:exhaustruct
		All: true,
		Filters: filters.NewArgs(filters.KeyValuePair{
			Key:   "label",
			Value: "percona.pmm.source=cli",
		}),
	})
}

// WaitForHealthyContainer waits until a containers is healthy.
func (b *Base) WaitForHealthyContainer(ctx context.Context, containerID string) <-chan WaitHealthyResponse {
	healthyChan := make(chan WaitHealthyResponse, 1)
	go func() {
		var res WaitHealthyResponse
		t := time.NewTicker(time.Second)
		defer t.Stop()

		for {
			logrus.Info("Checking if container is healthy...")
			status, err := b.Cli.ContainerInspect(ctx, containerID)
			if err != nil {
				res.Error = err
				break
			}

			if status.State == nil || status.State.Health == nil || status.State.Health.Status == "healthy" {
				res.Healthy = true
				break
			}

			select {
			case <-ctx.Done():
				healthyChan <- WaitHealthyResponse{Error: ctx.Err()}
				return
			default:
			}

			<-t.C
		}

		healthyChan <- res
	}()

	return healthyChan
}
