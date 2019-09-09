// pmm-agent
// Copyright (C) 2018 Percona LLC
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

package config

import (
	"sync"

	"github.com/sirupsen/logrus"
	"google.golang.org/grpc/grpclog"
)

// gRPCLogger is a compatibility wrapper between logrus entry and gRPC logger interface.
type gRPCLogger struct {
	*logrus.Entry
}

// V reports whether verbosity level l is at least the requested verbose level.
func (v *gRPCLogger) V(l int) bool {
	// we don't need real implementation ATM
	return true
}

// override InfoXXX methods with TraceXXX to keep gRPC and logrus levels in sync
func (v *gRPCLogger) Info(args ...interface{})                 { v.Trace(args...) }
func (v *gRPCLogger) Infoln(args ...interface{})               { v.Traceln(args...) }
func (v *gRPCLogger) Infof(format string, args ...interface{}) { v.Tracef(format, args...) }

var initLogger sync.Once

// ConfigureLogger configures standard Logrus logger.
func ConfigureLogger(cfg *Config) {
	initLogger.Do(func() {
		if cfg.Debug {
			logrus.SetLevel(logrus.DebugLevel)
		}

		if cfg.Trace {
			logrus.SetLevel(logrus.TraceLevel)

			// grpclog.SetLoggerV2 is not thread-safe
			grpclog.SetLoggerV2(&gRPCLogger{Entry: logrus.WithField("component", "grpclog")})

			// logrus.SetReportCaller not thread-safe: https://github.com/sirupsen/logrus/issues/954
			logrus.SetReportCaller(true)
		}
	})

	// logrus.GetLevel/SetLevel are thread-safe, so enable changing level without full restart,
	// and warn if other settings should be changed, but can't
	level := logrus.InfoLevel
	if cfg.Debug {
		level = logrus.DebugLevel
	}
	if cfg.Trace {
		level = logrus.TraceLevel
	}
	if logrus.GetLevel() != level {
		logrus.Warn("Some logging settings (caller reporter, gRPC tracing) can't be changed without restart.")
		logrus.SetLevel(level)
	}
}
