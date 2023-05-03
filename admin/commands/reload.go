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

package commands

import (
	"time"

	"github.com/percona/pmm/api/agentlocalpb/json/client"
	"github.com/percona/pmm/api/agentlocalpb/json/client/agent_local"
)

var reloadResultT = ParseTemplate(`
Reloaded.
`)

type reloadResult struct{}

func (res *reloadResult) Result() {}

func (res *reloadResult) String() string {
	return RenderTemplate(reloadResultT, res)
}

// ReloadCommand is used by Kong for CLI flags and commands.
type ReloadCommand struct {
	Timeout time.Duration `name:"wait" help:"Time to wait for a successful response from pmm-agent"`
}

// BeforeApply is run before the command is applied.
func (cmd *ReloadCommand) BeforeApply() error {
	return nil
}

// RunCmd runs the ReloadCommand.
func (cmd *ReloadCommand) RunCmd() (Result, error) {
	_, err := client.Default.AgentLocal.Reload(&agent_local.ReloadParams{
		Context: Ctx,
	})
	if err != nil {
		return nil, err
	}

	return &reloadResult{}, nil
}
