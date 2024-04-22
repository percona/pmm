// Copyright (C) 2024 Percona LLC
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

package interceptors

import (
	"context"
	"strings"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type serviceEnabled interface {
	Enabled() bool
}

// UnaryServiceEnabledInterceptor returns a new unary server interceptor that checks if service is enabled.
//
// Request on disabled service will be rejected with `FailedPrecondition` before reaching any userspace handlers.
func UnaryServiceEnabledInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		if svc, ok := info.Server.(serviceEnabled); ok && !svc.Enabled() {
			return nil, status.Errorf(codes.FailedPrecondition, "Service %s is disabled.", extractServiceName(info.FullMethod))
		}
		return handler(ctx, req)
	}
}

// StreamServiceEnabledInterceptor returns a new stream server interceptor that checks if service is enabled.
func StreamServiceEnabledInterceptor() grpc.StreamServerInterceptor {
	return func(srv interface{}, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		if svc, ok := srv.(serviceEnabled); ok && !svc.Enabled() {
			return status.Errorf(codes.FailedPrecondition, "Service %s is disabled.", extractServiceName(info.FullMethod))
		}
		return handler(srv, stream)
	}
}

func extractServiceName(fullMethod string) string {
	split := strings.Split(fullMethod, "/")
	if len(split) < 2 {
		return fullMethod
	}
	return split[1]
}
