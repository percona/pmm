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

	grpc_prometheus "github.com/grpc-ecosystem/go-grpc-prometheus"
	"google.golang.org/grpc/metadata"
)

type grpcCallerOrigin struct{}

type CallerOrigin string

const (
	InternalCallerOrigin = CallerOrigin("internal")
	ExternalCallerOrigin = CallerOrigin("external")
)

type GRPCMetricsExtension struct {
	grpc_prometheus.NullExtension
}

func (GRPCMetricsExtension) ServerHandledCounterCustomLabels() []string {
	return []string{"caller_origin"}
}

func (GRPCMetricsExtension) ServerHandledCounterPreRegisterValues() [][]string {
	return [][]string{
		{string(InternalCallerOrigin)},
		{string(ExternalCallerOrigin)},
	}
}

func (GRPCMetricsExtension) ServerHandledCounterValues(ctx context.Context) []string {
	return []string{GetCallerOriginStr(ctx)}
}

func SetCallerOrigin(ctx context.Context, method string) context.Context {
	return context.WithValue(ctx, grpcCallerOrigin{}, callerOriginFromRequest(ctx, method))
}

func GetCallerOriginStr(ctx context.Context) string {
	value, ok := ctx.Value(grpcCallerOrigin{}).(CallerOrigin)
	if ok {
		return string(value)
	}

	return ""
}

func callerOriginFromRequest(ctx context.Context, method string) CallerOrigin {
	if method == "/server.Server/Readiness" || method == "/agent.Agent/Connect" {
		return InternalCallerOrigin
	}

	headers, _ := metadata.FromIncomingContext(ctx)

	// if referer is present - the caller is an external one
	if len(headers.Get("grpcgateway-referer")) == 0 {
		return ExternalCallerOrigin
	}

	return InternalCallerOrigin
}
