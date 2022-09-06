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

// Package docker stores common functions for working with Docker
package docker

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
	"github.com/sirupsen/logrus"
)

func IsDockerInstalled() (bool, error) {
	path, err := exec.LookPath("docker")
	if err != nil {
		return false, err
	}

	logrus.Debugf("Found docker in %s", path)

	return true, nil
}

func downloadDockerInstallScript() (io.ReadCloser, error) {
	res, err := http.Get("https://get.docker.com/")
	if err != nil {
		return nil, err
	}

	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("received HTTP %d when downloading Docker install script", res.StatusCode)
	}

	return res.Body, nil
}

func InstallDocker() error {
	script, err := downloadDockerInstallScript()
	if err != nil {
		return err
	}

	cmd := exec.Command("sh", "-s")
	cmd.Stdin = script
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err = cmd.Run()
	if err != nil {
		return err
	}

	return nil
}

func GetDockerClient(ctx context.Context) (*client.Client, error) {
	cli, err := client.NewClientWithOpts(client.WithVersion("1.41"))

	return cli, err
}

func FindServerContainers(ctx context.Context, cli *client.Client) ([]types.Container, error) {
	return cli.ContainerList(ctx, types.ContainerListOptions{
		All: true,
		Filters: filters.NewArgs(filters.KeyValuePair{
			Key:   "label",
			Value: "percona.pmm=server",
		}),
	})
}

func ChangeServerPassword(ctx context.Context, cli *client.Client, containerID, newPassword string) error {
	logrus.Info("Changing password")

	exec, err := cli.ContainerExecCreate(ctx, containerID, types.ExecConfig{
		Cmd:          []string{"change-admin-password", newPassword},
		Tty:          true,
		AttachStderr: true,
		AttachStdout: true,
	})
	if err != nil {
		return err
	}

	err = cli.ContainerExecStart(ctx, exec.ID, types.ExecStartCheck{})
	if err != nil {
		return err
	}

	logrus.Info("Changed password")

	return nil
}

type WaitHealthyResponse struct {
	Healthy bool
	Error   error
}

func WaitForHealthyContainer(ctx context.Context, cli *client.Client, containerID string) <-chan WaitHealthyResponse {
	healthyChan := make(chan WaitHealthyResponse, 1)
	go func() {
		var res WaitHealthyResponse
		t := time.NewTicker(time.Second)
		defer t.Stop()

		for {
			logrus.Info("Checking if container is healthy...")
			status, err := cli.ContainerInspect(ctx, containerID)
			if err != nil {
				res.Error = err
				break
			}

			if status.State.Health.Status == "healthy" {
				res.Healthy = true
				break
			}

			<-t.C
		}

		healthyChan <- res
	}()

	return healthyChan
}
