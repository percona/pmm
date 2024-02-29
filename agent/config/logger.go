// Copyright (C) 2023 Percona LLC
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
func (v *gRPCLogger) V(l int) bool { //nolint:revive
	// we don't need real implementation ATM
	return true
}

// override InfoXXX methods with TraceXXX to keep gRPC and logrus levels in sync.
func (v *gRPCLogger) Info(args ...interface{})                 { v.Trace(args...) }
func (v *gRPCLogger) Infoln(args ...interface{})               { v.Traceln(args...) }
func (v *gRPCLogger) Infof(format string, args ...interface{}) { v.Tracef(format, args...) }

var initLogger sync.Once

// ConfigureLogger configures standard Logrus logger.
func ConfigureLogger(cfg *Config) {
	level := parseLoggerConfig(cfg.LogLevel, cfg.Debug, cfg.Trace)

	initLogger.Do(func() {
		logrus.SetLevel(level)

		if level == logrus.TraceLevel {
			// grpclog.SetLoggerV2 is not thread-safe
			grpclog.SetLoggerV2(&gRPCLogger{Entry: logrus.WithField("component", "grpclog")})

			// logrus.SetReportCaller thread-safe: https://github.com/sirupsen/logrus/issues/954
			logrus.SetReportCaller(true)
		}
	})

	// logrus.GetLevel/SetLevel are thread-safe, so enable changing level without full restart,
	// and warn if other settings should be changed, but can't
	if logrus.GetLevel() != level {
		logrus.Warn("Some logging settings (gRPC tracing) can't be changed without restart.")

		logrus.SetLevel(level)

		if level == logrus.TraceLevel {
			// logrus.SetReportCaller thread-safe: https://github.com/sirupsen/logrus/issues/954
			logrus.SetReportCaller(true)
		}
	}
}

func parseLoggerConfig(level string, debug, trace bool) logrus.Level {
	if trace {
		return logrus.TraceLevel
	}

	if debug {
		return logrus.DebugLevel
	}

	if level != "" {
		parsedLevel, err := logrus.ParseLevel(level)
		if err != nil {
			logrus.Errorf("config: cannot parse logging level: %s, error: %v", level, err)
		} else {
			return parsedLevel
		}
	}

	// info level set by default, because we use info level to write logs of exporters.
	return logrus.InfoLevel
}
