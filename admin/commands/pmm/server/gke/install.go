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

package gke

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"time"

	"github.com/percona/pmm/admin/cli/flags"
	"github.com/percona/pmm/admin/commands"
)

// InstallCommand is used by Kong for CLI flags and commands.
type InstallCommand struct{}

type installResult struct{}

// Result is a command run result.
func (res *installResult) Result() {}

// String stringifies command result.
func (res *installResult) String() string {
	return "works"
}

// RunCmdWithContext runs install command.
func (c *InstallCommand) RunCmdWithContext(ctx context.Context, flags *flags.GlobalFlags) (commands.Result, error) {
	start := time.Now()
	cmd := exec.Command(
		"gcloud",
		"container",
		"clusters",
		"create",
		"--zone europe-west1-b",
		"pmm-dbaas-cluster",
		"--machine-type e2-standard-4",
		"--preemptible",
		"--num-nodes=3",
	)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return nil, err
	}

	cmd = exec.Command(
		"gcloud",
		"container",
		"clusters",
		"get-credentials",
		"pmm-dbaas-cluster",
		"--zone=europe-west3-c",
	)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return nil, err
	}

	cmd = exec.Command("kubectl", "apply", "-f", "pmm-server.yaml")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return nil, err
	}

	fmt.Printf("Elapsed time %s\n", time.Since(start))

	return &installResult{}, nil
}
