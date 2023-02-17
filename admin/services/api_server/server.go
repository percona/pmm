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

// Package api_server holds logic for operating API server.
package api_server

import (
	"context"
	"errors"
	"net"
	"os"
	"time"

	channelz "google.golang.org/grpc/channelz/service"

	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	grpc_validator "github.com/grpc-ecosystem/go-grpc-middleware/validator"
	"github.com/percona/pmm/admin/services/status"
	"github.com/percona/pmm/admin/services/update"
	"github.com/percona/pmm/api/updatepb"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

const (
	gRPCMessageMaxSize = 100 * 1024 * 1024
	socketPath         = "/srv/pmm-server-upgrade.sock"
	shutdownTimeout    = 1 * time.Second
)

type Server struct {
	EnableDebug bool
	Update      UpdateOpts
}

type UpdateOpts struct {
	DockerImage string
}

func New(dockerImage string) *Server {
	return &Server{
		Update: UpdateOpts{
			DockerImage: dockerImage,
		},
	}
}

func (s *Server) Run(ctx context.Context) {
	l := logrus.WithField("component", "local-server/gRPC")

	gRPCServer := grpc.NewServer(
		grpc.MaxRecvMsgSize(gRPCMessageMaxSize),
		grpc.UnaryInterceptor(grpc_middleware.ChainUnaryServer(
			grpc_validator.UnaryServerInterceptor(),
		)),
	)

	u, err := update.New(ctx, s.Update.DockerImage, gRPCMessageMaxSize)
	if err != nil {
		l.Fatal(err)
	}

	updatepb.RegisterStatusServer(gRPCServer, status.New())
	updatepb.RegisterUpdateServer(gRPCServer, u)

	if s.EnableDebug {
		l.Debug("Reflection and channelz are enabled")
		reflection.Register(gRPCServer)
		channelz.RegisterChannelzServiceToServer(gRPCServer)
	}

	// run server until it is stopped gracefully
	go func() {
		var err error
		for {
			l.Infof("Starting gRPC server on unix://%s", socketPath)
			err = os.Remove(socketPath)
			if err != nil && !errors.Is(err, os.ErrNotExist) {
				l.Panic(err)
			}

			var listener net.Listener
			listener, err = net.Listen("unix", socketPath)
			if err != nil {
				l.Panic(err)
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
