// Copyright (C) 2023 Percona LLC
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

	grpc_prometheus "github.com/grpc-ecosystem/go-grpc-prometheus"
	"google.golang.org/grpc/metadata"
)

type grpcCallerOrigin struct{}

type callerOrigin string

const (
	internalCallerOrigin = callerOrigin("internal")
	externalCallerOrigin = callerOrigin("external")
)

// GRPCMetricsExtension for extra labels in /debug/metrics.
type GRPCMetricsExtension struct { //nolint:recvcheck
	grpc_prometheus.DefaultExtension
}

// MetricsNameAdjust adjusts the given metric name and returns the adjusted name.
func (e GRPCMetricsExtension) MetricsNameAdjust(name string) string {
	return "pmm_" + name
}

// ServerStreamMsgReceivedCounterCustomLabels returns custom labels for the server stream message received counter.
func (e *GRPCMetricsExtension) ServerStreamMsgReceivedCounterCustomLabels() []string {
	return []string{"caller_origin"}
}

// ServerStreamMsgReceivedCounterValues returns custom values for the server stream message received counter.
func (e *GRPCMetricsExtension) ServerStreamMsgReceivedCounterValues(ctx context.Context) []string {
	return []string{getCallerOriginStr(ctx)}
}

var _ grpc_prometheus.ServerExtension = &GRPCMetricsExtension{}

// SetCallerOrigin returns derived context with metric.
func SetCallerOrigin(ctx context.Context, method string) context.Context {
	return context.WithValue(ctx, grpcCallerOrigin{}, callerOriginFromRequest(ctx, method))
}

func getCallerOriginStr(ctx context.Context) string {
	v := ctx.Value(grpcCallerOrigin{})
	if v == nil {
		return ""
	}
	return string(v.(callerOrigin)) //nolint:forcetypeassert
}

func callerOriginFromRequest(ctx context.Context, method string) callerOrigin {
	if method == "/server.v1.ServerService/Readiness" || method == "/agent.v1.AgentService/Connect" {
		return internalCallerOrigin
	}

	headers, _ := metadata.FromIncomingContext(ctx)

	// if referer is present - the caller is an external one
	if len(headers.Get("grpcgateway-referer")) == 0 {
		return externalCallerOrigin
	}

	return internalCallerOrigin
}
