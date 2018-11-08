// pmm-managed
// Copyright (C) 2017 Percona LLC
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program. If not, see <https://www.gnu.org/licenses/>.

// Package interceptors contains gRPC wrappers for logging and Prometheus metrics.
package interceptors

import (
	"context"

	middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	"github.com/grpc-ecosystem/go-grpc-prometheus"
	"google.golang.org/grpc"

	"github.com/percona/pmm-managed/utils/logger"
)

// Unary adds context logger and Prometheus metrics to unary server RPC.
func Unary(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	ctx, _ = logger.Set(ctx, logger.MakeRequestID())
	return grpc_prometheus.UnaryServerInterceptor(ctx, req, info, handler)
}

// Stream adds context logger and Prometheus metrics to stream server RPC.
func Stream(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
	wrapped := middleware.WrapServerStream(ss)
	wrapped.WrappedContext, _ = logger.Set(ss.Context(), logger.MakeRequestID())

	return grpc_prometheus.StreamServerInterceptor(srv, wrapped, info, handler)
}

// check interfaces
var (
	_ grpc.UnaryServerInterceptor  = Unary
	_ grpc.StreamServerInterceptor = Stream
)
