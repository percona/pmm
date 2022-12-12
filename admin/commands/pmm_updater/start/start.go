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

// Package start holds logic for starting pmm-updater.
package start

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/network"
	"github.com/sirupsen/logrus"

	"github.com/percona/pmm/admin/cli/flags"
	"github.com/percona/pmm/admin/commands"
	"github.com/percona/pmm/admin/commands/base"
	dockerCmd "github.com/percona/pmm/admin/commands/pmm/server/docker"
	"github.com/percona/pmm/admin/pkg/docker"
	serverpb "github.com/percona/pmm/api/serverpb/json/client"
	"github.com/percona/pmm/api/serverpb/json/client/server"
)

// StartCommand is used by Kong for CLI flags and commands.
type StartCommand struct {
	DockerNetworkName string        `default:"pmm-updater" help:"Network name in Docker to be used to connect PMM Updater and PMM Server instances"`
	DockerImage       string        `default:"percona/pmm-server:2" help:"Docker image to use for updating to the latest version"`
	WaitBetweenChecks time.Duration `name:"wait" default:"60s" help:"Time duration to wait between checking for updates"`

	dockerFn Functions
	globals  *flags.GlobalFlags
}

type startResult struct{}

// Result is a command run result.
func (r *startResult) Result() {}

// String stringifies command result.
func (r *startResult) String() string {
	return "ok"
}

// BeforeApply is run before the command is applied.
func (cmd *StartCommand) BeforeApply() error {
	commands.SetupClientsEnabled = false
	return nil
}

// RunCmdWithContext runs command
func (c *StartCommand) RunCmdWithContext(ctx context.Context, globals *flags.GlobalFlags) (commands.Result, error) {
	logrus.Info("Starting updater")

	c.globals = globals

	if c.dockerFn == nil {
		d, err := docker.New(nil)
		if err != nil {
			return nil, err
		}

		c.dockerFn = d
	}

	if !c.dockerFn.HaveDockerAccess(ctx) {
		return nil, fmt.Errorf("cannot access Docker. Make sure this container has access to the docker socket")
	}

	logrus.Info("Initializing network")
	if err := c.initDockerNetwork(ctx); err != nil {
		return nil, err
	}

	if err := c.runUpdateCheckLoop(ctx); err != nil {
		return nil, err
	}

	return &startResult{}, nil
}

func (c *StartCommand) initDockerNetwork(ctx context.Context) error {
	net, err := c.dockerFn.NetworkInspect(ctx, c.DockerNetworkName, types.NetworkInspectOptions{})
	if err != nil {
		if c.dockerFn.IsErrNotFound(err) {
			err = c.createNetwork(ctx)
		}

		if err != nil {
			return err
		}
	}

	selfName, err := os.Hostname()
	if err != nil {
		return err
	}

	return c.connectContainerToNetwork(ctx, net, selfName)
}

// connectContainerToNetwork connects container to a network. It's a no-op if the container is already connected.
func (c *StartCommand) connectContainerToNetwork(ctx context.Context, net types.NetworkResource, containerID string) error {
	found := false
	for contID := range net.Containers {
		if strings.HasPrefix(contID, containerID) {
			found = true
		}
	}

	if found {
		logrus.Debugf("Container %s already connected to network %s", containerID, net.Name)
		return nil
	}

	return c.dockerFn.NetworkConnect(ctx, c.DockerNetworkName, containerID, &network.EndpointSettings{})
}

func (c *StartCommand) createNetwork(ctx context.Context) error {
	_, err := c.dockerFn.NetworkCreate(ctx, c.DockerNetworkName, types.NetworkCreate{
		Driver: "bridge",
	})

	return err
}

func (c *StartCommand) runUpdateCheckLoop(ctx context.Context) error {
	for {
		logrus.Info("Checking update requests")

		if err := c.checkForUpdateRequest(ctx); err != nil {
			logrus.Error(err)
		}

		// Sleep for a bit
		logrus.Infof("Sleeping for %s before next update check", c.WaitBetweenChecks)
		select {
		case <-time.After(c.WaitBetweenChecks):
		case <-ctx.Done():
			return nil
		}
	}
}

func (c *StartCommand) checkForUpdateRequest(ctx context.Context) error {
	containers, err := c.dockerFn.FindServerContainers(ctx)
	if err != nil {
		return err
	}

	net, err := c.dockerFn.NetworkInspect(ctx, c.DockerNetworkName, types.NetworkInspectOptions{})
	if err != nil {
		return err
	}

	logrus.Debugf("Found %d containers with PMM Server", len(containers))

	for _, cont := range containers {
		if cont.State != "running" {
			logrus.Debugf("Container %s it not running. Skipping.", cont.ID)
			continue
		}
		logrus.Debugf("Connecting container %s to network", cont.ID)
		if err := c.connectContainerToNetwork(ctx, net, cont.ID); err != nil {
			logrus.Errorf("Could not connect container %s to updater network. Error: %v", cont.ID, err)
			continue
		}

		logrus.Debugf("Inspecting container %s", cont.ID)
		cInspect, err := c.dockerFn.ContainerInspect(ctx, cont.ID)
		if err != nil {
			logrus.Errorf("Could not inspect container %s. Error: %v", cont.ID, err)
			continue
		}

		logrus.Debugf("Checking if update is requested for container %s with hostname %q", cont.ID, cInspect.Config.Hostname)
		isUpdateRequested, err := c.isUpdateRequested(ctx, cInspect.Config.Hostname)
		if err != nil {
			logrus.Errorf("Cannot check if update is requested for container %s. Error %v", cont.ID, err)
			continue
		}

		if isUpdateRequested {
			logrus.Debugf("Starting upgrade for container %s", cont.ID)
			cmd := &dockerCmd.UpgradeCommand{
				ContainerID:            cont.ID,
				DockerImage:            c.DockerImage,
				NewContainerNamePrefix: "pmm-server",
			}

			_, err := cmd.RunCmdWithContext(ctx, c.globals)
			if err != nil {
				logrus.Errorf("Could not upgrade container %s. Error: %v", cont.ID, err)
				continue
			}
		}
	}

	return nil
}

func (c *StartCommand) isUpdateRequested(ctx context.Context, hostname string) (bool, error) {
	u, err := url.Parse(fmt.Sprintf("http://%s/", hostname))
	if err != nil {
		return false, err
	}

	transport := base.GetGRPCTransport(
		ctx, u, c.globals.EnableDebug || c.globals.EnableTrace, true,
		logrus.Fields{
			"component": "server-transport",
			"host":      u.Host,
		})

	serverAPI := serverpb.New(transport, nil)

	status, err := serverAPI.Server.SideContainerUpdateStatus(&server.SideContainerUpdateStatusParams{Context: ctx})
	if err != nil {
		return false, err
	}

	return status.Payload.IsRequested, nil
}
