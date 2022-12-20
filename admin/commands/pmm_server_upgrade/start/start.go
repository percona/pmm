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

// Package start holds logic for starting pmm-server-upgrade.
package start

import (
	"context"
	"fmt"
	"net"
	"os"
	"time"

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
	socketPath         = "/srv/pmm-server-upgrade.sock"
	shutdownTimeout    = 1 * time.Second
)

// StartCommand is used by Kong for CLI flags and commands.
type StartCommand struct {
	DockerImage string `default:"percona/pmm-server:2" help:"Docker image to use for updating to the latest version"`

	dockerFn functions
	globals  *flags.GlobalFlags
}

type startResult struct{}

// Result is a command run result.
func (r *startResult) Result() {}

// String stringifies command result.
func (r *startResult) String() string {
	return "Exiting"
}

// BeforeApply is run before the command is applied.
func (c *StartCommand) BeforeApply() error {
	commands.SetupClientsEnabled = false
	return nil
}

// RunCmdWithContext runs command
func (c *StartCommand) RunCmdWithContext(ctx context.Context, globals *flags.GlobalFlags) (commands.Result, error) {
	logrus.Info("Starting PMM Server Upgrade")

	c.globals = globals

	if c.dockerFn == nil {
		d, err := docker.New(nil)
		if err != nil {
			return nil, err
		}

		c.dockerFn = d
	}

	if !c.dockerFn.HaveDockerAccess(ctx) {
		return nil, fmt.Errorf("cannot access Docker. Make sure this container has access to the Docker socket")
	}

	c.runAPIServer(ctx)

	return &startResult{}, nil
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
		l.Debug("Reflection and channelz are enabled")
		reflection.Register(gRPCServer)
		channelz.RegisterChannelzServiceToServer(gRPCServer)
	}

	// run server until it is stopped gracefully
	go func() {
		var err error
		for {
			l.Infof("Starting gRPC server on unix://%s", socketPath)
			err := os.Remove(socketPath)
			if err != nil && !errors.Is(err, os.ErrNotExist) {
				logrus.Panic(err)
			}

			listener, err := net.Listen("unix", socketPath)
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
		l.Debug("Server stopped")
	}()

	<-ctx.Done()

	// Try to stop server gracefully or force the stop after a timeout.
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
