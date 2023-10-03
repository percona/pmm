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
	"runtime/debug"
	"runtime/pprof"
	"time"

	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	grpc_prometheus "github.com/grpc-ecosystem/go-grpc-prometheus"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/percona/pmm/qan-api2/utils/logger"
)

const responseTimeout = 2 * time.Minute

func logRequest(l *logrus.Entry, prefix string, f func() error) (err error) {
	start := time.Now()
	l.Infof("Starting %s ...", prefix)

	defer func() {
		dur := time.Since(start)

		if p := recover(); p != nil {
			// Always log with %+v, even before re-panic - there can be inner stacktraces
			// produced by panic(errors.WithStack(err)).
			// Also always log debug.Stack() for all panics.
			l.Errorf("%s done in %s with panic: %+v\nStack: %s", prefix, dur, p, debug.Stack())

			if l.Logger.GetLevel() == logrus.TraceLevel {
				panic(p)
			}

			err = status.Error(codes.Internal, "Internal server error.")
			return
		}

		// log gRPC errors as warning, not errors, even if they are wrapped
		_, gRPCError := status.FromError(errors.Cause(err))
		switch {
		case err == nil:
			if dur < time.Second {
				l.Infof("%s done in %s.", prefix, dur)
			} else {
				l.Warnf("%s done in %s (quite long).", prefix, dur)
			}
		case gRPCError:
			// %+v for inner stacktraces produced by errors.WithStack(err)
			l.Warnf("%s done in %s with gRPC error: %+v", prefix, dur, err)
		default:
			// %+v for inner stacktraces produced by errors.WithStack(err)
			l.Errorf("%s done in %s with unexpected error: %+v", prefix, dur, err)
			err = status.Error(codes.Internal, "Internal server error.")
		}
	}()

	err = f()
	return //nolint:nakedret
}

// Unary adds context logger and Prometheus metrics to unary server RPC.
func Unary(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	ctx, cancel := context.WithTimeout(ctx, responseTimeout)
	defer cancel()

	// add pprof labels for more useful profiles
	defer pprof.SetGoroutineLabels(ctx)
	ctx = pprof.WithLabels(ctx, pprof.Labels("method", info.FullMethod))
	pprof.SetGoroutineLabels(ctx)

	// set logger
	l := logrus.WithField("request", logger.MakeRequestID())
	ctx = logger.SetEntry(ctx, l)

	var res interface{}
	err := logRequest(l, "RPC "+info.FullMethod, func() error {
		var origErr error
		res, origErr = grpc_prometheus.UnaryServerInterceptor(ctx, req, info, handler)
		l.Debugf("\nRequest:\n%s\nResponse:\n%s\n", req, res)
		return origErr
	})
	return res, err
}

// Stream adds context logger and Prometheus metrics to stream server RPC.
func Stream(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
	ctx := ss.Context()

	// add pprof labels for more useful profiles
	defer pprof.SetGoroutineLabels(ctx)
	ctx = pprof.WithLabels(ctx, pprof.Labels("method", info.FullMethod))
	pprof.SetGoroutineLabels(ctx)

	// set logger
	l := logrus.WithField("request", logger.MakeRequestID())
	ctx = logger.SetEntry(ctx, l)

	err := logRequest(l, "Stream "+info.FullMethod, func() error {
		wrapped := grpc_middleware.WrapServerStream(ss)
		wrapped.WrappedContext = ctx
		return grpc_prometheus.StreamServerInterceptor(srv, wrapped, info, handler)
	})
	return err
}

// check interfaces.
var (
	_ grpc.UnaryServerInterceptor  = Unary
	_ grpc.StreamServerInterceptor = Stream
)
