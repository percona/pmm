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
	"os"
	"os/exec"
	"time"

	"github.com/sirupsen/logrus"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/container/v1"

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

	logrus.Info("Creating GKE")

	cl, err := google.DefaultClient(ctx, container.CloudPlatformScope)
	if err != nil {
		return nil, err
	}

	containerService, err := container.New(cl)
	if err != nil {
		return nil, err
	}

	op, err := createGKECluster(ctx, containerService)
	if err != nil {
		return nil, err
	}

	ch := make(chan struct{})
	go func() {
		defer close(ch)
		t := time.NewTicker(5 * time.Second)
		name := "projects/percona-pmm-dev/locations/europe-west-1/operations/"

		for {
			<-t.C
			op, err := containerService.Projects.Locations.Operations.Get(name + op.Name).Context(ctx).Do()
			if err != nil {
				logrus.Info(err)
			}

			if op.Status == "DONE" {
				return
			}

			logrus.Info(op.Progress.Metrics)
		}
	}()

	<-ch

	logrus.Infof("Elapsed time %s\n", time.Since(start))
	logrus.Info("Getting credentials")
	cmd := exec.Command(
		"gcloud",
		"container",
		"clusters",
		"get-credentials",
		"michal-dbaas",
		"--zone=europe-west1-b",
	)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return nil, err
	}

	logrus.Infof("Elapsed time %s\n", time.Since(start))
	logrus.Info("Running kubectl")
	cmd = exec.Command("kubectl", "apply", "-f", "/home/michal/pmm-server.yaml")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return nil, err
	}

	logrus.Infof("Elapsed time %s\n", time.Since(start))

	return &installResult{}, nil
}

func createGKECluster(ctx context.Context, containerService *container.Service) (*container.Operation, error) {
	parent := "projects/percona-pmm-dev/locations/europe-west-1"

	rb := &container.CreateClusterRequest{
		Cluster: &container.Cluster{
			Zone:             "europe-west1-b",
			InitialNodeCount: 3,
			NodeConfig: &container.NodeConfig{
				Preemptible: true,
				MachineType: "e2-standard-4",
			},
		},
	}

	resp, err := containerService.Projects.Locations.Clusters.Create(parent, rb).Context(ctx).Do()
	if err != nil {
		return nil, err
	}

	return resp, nil
}
