// pmm-agent
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
