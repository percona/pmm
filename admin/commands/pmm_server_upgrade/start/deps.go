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

package start

import (
	"context"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
)

//go:generate ../../../../bin/mockery -name=functions -case=snake -inpkg -testonly

// containerManager contain methods required to interact with Docker containers.
type containerManager interface {
	ContainerList(ctx context.Context, options types.ContainerListOptions) ([]types.Container, error)
	ContainerInspect(ctx context.Context, containerID string) (types.ContainerJSON, error)
	GetDockerClient() *client.Client
	HaveDockerAccess(ctx context.Context) bool
	FindServerContainers(ctx context.Context) ([]types.Container, error)
}
