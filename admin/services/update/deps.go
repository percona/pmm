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

package update

import (
	"context"

	"github.com/docker/docker/api/types"
)

//go:generate ../../../bin/mockery -name=dockerFunctions -case=snake -inpkg -testonly

// functions contain methods required to interact with Docker.
type dockerFunctions interface {
	ContainerInspect(ctx context.Context, containerID string) (types.ContainerJSON, error)
}
