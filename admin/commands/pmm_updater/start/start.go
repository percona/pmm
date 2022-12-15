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
	"net"
	"os"
	"strings"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/network"
	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	grpc_validator "github.com/grpc-ecosystem/go-grpc-middleware/validator"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	channelz "google.golang.org/grpc/channelz/service"
	"google.golang.org/grpc/reflection"

	"github.com/percona/pmm/admin/cli/flags"
	"github.com/percona/pmm/admin/commands"
	"github.com/percona/pmm/admin/pkg/docker"
	"github.com/percona/pmm/admin/services/status"
	"github.com/percona/pmm/admin/services/update"
	"github.com/percona/pmm/api/updatepb"
)

const (
	gRPCMessageMaxSize = 100 * 1024 * 1024
	shutdownTimeout    = 1 * time.Second
)

// StartCommand is used by Kong for CLI flags and commands.
type StartCommand struct {
	DockerNetworkName string        `default:"pmm-updater" help:"Network name in Docker to be used to connect PMM Updater and PMM Server instances"`
	DockerImage       string        `default:"percona/pmm-server:2" help:"Docker image to use for updating to the latest version"`
	WaitBetweenChecks time.Duration `name:"wait" default:"60s" help:"Time duration to wait between checks for updates"`

	dockerFn functions
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
func (c *StartCommand) BeforeApply() error {
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

	c.runAPIServer(ctx)

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

func (c *StartCommand) runAPIServer(ctx context.Context) {
	l := logrus.WithField("component", "local-server/gRPC")

	gRPCServer := grpc.NewServer(
		grpc.MaxRecvMsgSize(gRPCMessageMaxSize),
		grpc.UnaryInterceptor(grpc_middleware.ChainUnaryServer(
			grpc_validator.UnaryServerInterceptor(),
		)),
	)

	u, err := update.New(ctx, c.DockerImage, gRPCMessageMaxSize)
	if err != nil {
		logrus.Fatal(err)
	}

	updatepb.RegisterStatusServer(gRPCServer, status.New())
	updatepb.RegisterUpdateServer(gRPCServer, u)

	if c.globals.EnableDebug {
		l.Debug("Reflection and channelz are enabled.")
		reflection.Register(gRPCServer)
		channelz.RegisterChannelzServiceToServer(gRPCServer)
	}

	// run server until it is stopped gracefully or not
	go func() {
		var err error
		for {
			l.Infof("Starting gRPC server on unix:///srv/pmm-updater.sock")

			listener, err := net.Listen("unix", "/srv/pmm-updater.sock")
			if err != nil {
				logrus.Panic(err)
			}

			err = gRPCServer.Serve(listener) // listener will be closed when this method returns
			if err == nil || errors.Is(err, grpc.ErrServerStopped) {
				break
			}
		}
		if err != nil {
			l.Errorf("Failed to serve: %s", err)
			return
		}
		l.Debug("Server stopped.")
	}()

	<-ctx.Done()

	// Try to stop server gracefully and force the stop after a timeout.
	stopped := make(chan struct{})
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), shutdownTimeout)
	go func() {
		<-shutdownCtx.Done()
		gRPCServer.Stop()
		close(stopped)
	}()
	gRPCServer.GracefulStop()
	shutdownCancel()
	<-stopped
}
