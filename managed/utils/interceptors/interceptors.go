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

package interceptors

import (
	"context"
	"database/sql/driver"
	"errors"
	"net"
	"regexp"
	"runtime/debug"
	"runtime/pprof"
	"time"

	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	"github.com/lib/pq"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/prototext"
	"google.golang.org/protobuf/proto"

	agentv1 "github.com/percona/pmm/api/agent/v1"
	"github.com/percona/pmm/utils/logger"
)

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
		_, gRPCError := status.FromError(err)
		switch {
		case err == nil:
			if dur < time.Second {
				l.Infof("%s done in %s.", prefix, dur)
			} else {
				l.Warnf("%s done in %s (quite long).", prefix, dur)
			}
		case gRPCError:
			l.Warnf("%s done in %s with gRPC error: %+v", prefix, dur, err)
		case isDatabaseUnavailable(err):
			l.Errorf("%s done in %s with database error: %+v", prefix, dur, err)
			err = status.Error(codes.Unavailable, "Database is unavailable, please retry.")
		default:
			l.Errorf("%s done in %s with unexpected error: %+v", prefix, dur, err)
			err = status.Error(codes.Internal, "Internal server error.")
		}
	}()

	err = f()
	return err
}

// isDatabaseUnavailable reports whether err means the database could not serve the request at all,
// so nothing was committed and the client may safely retry (e.g. during a PostgreSQL failover in HA).
func isDatabaseUnavailable(err error) bool {
	pqErr, ok := errors.AsType[*pq.Error](err)
	if ok {
		switch pqErr.Code {
		case "57P01", "57P02", "57P03": // admin_shutdown, crash_shutdown, cannot_connect_now
			return true
		default:
			return false
		}
	}

	opErr, ok := errors.AsType[*net.OpError](err)
	if ok && opErr.Op == "dial" {
		return true
	}

	return errors.Is(err, driver.ErrBadConn)
}

// UnaryInterceptorType represents the type of a unary gRPC interceptor.
type UnaryInterceptorType = func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error)

var dropEndpointsRE = regexp.MustCompile(`^/server.v1.ServerService/(Readiness|LeaderHealthCheck)$`)

// UnaryAdd adds context logger and Prometheus metrics to unary server RPC.
func UnaryAdd(interceptor grpc.UnaryServerInterceptor) UnaryInterceptorType {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		// add pprof labels for more useful profiles
		defer pprof.SetGoroutineLabels(ctx)
		ctx = pprof.WithLabels(ctx, pprof.Labels("method", info.FullMethod))
		pprof.SetGoroutineLabels(ctx)

		// set logger
		l := logrus.WithFields(logrus.Fields{
			"request":   logger.MakeRequestID(),
			"component": "grpc/interceptor",
			"method":    info.FullMethod,
		})
		ctx = logger.SetEntry(ctx, l)
		ctx = SetCallerOrigin(ctx, info.FullMethod)

		var res any
		err := logRequest(l, "RPC "+info.FullMethod, func() error {
			var origErr error
			res, origErr = interceptor(ctx, req, info, handler)
			if l.Logger.IsLevelEnabled(logrus.DebugLevel) && !dropEndpointsRE.MatchString(info.FullMethod) {
				var reqMsg, resMsg any
				protoReq, okReq := req.(proto.Message)
				if okReq {
					reqMsg = prototext.Format(logger.RedactMessage(protoReq))
				} else {
					reqMsg = req
				}

				protoResp, okResp := res.(proto.Message)
				if okResp {
					resMsg = prototext.Format(logger.RedactMessage(protoResp))
				} else {
					resMsg = res
				}
				l.Debugf("\nRequest:\n%s\nResponse:\n%s\n", reqMsg, resMsg)
			}
			return origErr
		})
		return res, err
	}
}

// Stream adds context logger and Prometheus metrics to stream server RPC.
func Stream(interceptor grpc.StreamServerInterceptor) func(srv any, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
	return func(srv any, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		ctx := ss.Context()

		// add pprof labels for more useful profiles
		defer pprof.SetGoroutineLabels(ctx)
		ctx = pprof.WithLabels(ctx, pprof.Labels("method", info.FullMethod))
		pprof.SetGoroutineLabels(ctx)

		// set logger
		l := logrus.WithFields(logrus.Fields{
			"request":   logger.MakeRequestID(),
			"component": "grpc/interceptor",
			"method":    info.FullMethod,
		})
		if info.FullMethod == "/agent.v1.AgentService/Connect" ||
			info.FullMethod == "/realtimeanalytics.v1.CollectorService/Collect" {
			md, _ := agentv1.ReceiveAgentConnectMetadata(ss)
			if md != nil && md.ID != "" {
				l = l.WithField("agent_id", md.ID)
			}
		}
		ctx = logger.SetEntry(ctx, l)

		ctx = SetCallerOrigin(ctx, info.FullMethod)

		err := logRequest(l, "Stream "+info.FullMethod, func() error {
			wrapped := grpc_middleware.WrapServerStream(ss)
			wrapped.WrappedContext = ctx
			return interceptor(srv, wrapped, info, handler)
		})
		return err
	}
}

// check interfaces.
var (
	_ grpc.UnaryServerInterceptor  = UnaryAdd(nil)
	_ grpc.StreamServerInterceptor = Stream(nil)
)
