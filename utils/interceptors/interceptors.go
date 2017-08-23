// pmm-managed
// Copyright (C) 2017 Percona LLC
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published
// by the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program. If not, see <http://www.gnu.org/licenses/>.

// Package interceptors contains gRPC wrappers for logging and Prometheus metrics.
package interceptors

import (
	"github.com/grpc-ecosystem/go-grpc-prometheus"
	"golang.org/x/net/context"
	"google.golang.org/grpc"

	"github.com/percona/pmm-managed/utils/logger"
)

// Unary adds context logger and Prometheus metrics to unary server RPC.
func Unary(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	ctx, _ = logger.Set(ctx, logger.MakeRequestID())
	return grpc_prometheus.UnaryServerInterceptor(ctx, req, info, handler)
}

// Stream adds Prometheus metrics to unary server RPC. Logger should be explicitly set by handler if required.
// See https://github.com/grpc-ecosystem/go-grpc-prometheus/issues/24
func Stream(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
	return grpc_prometheus.StreamServerInterceptor(srv, ss, info, handler)
}

// check interfaces
var (
	_ grpc.UnaryServerInterceptor  = Unary
	_ grpc.StreamServerInterceptor = Stream
)
