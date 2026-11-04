// Copyright (C) 2023 Percona LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package logger

import (
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc/grpclog"
)

// GRPC is a compatibility wrapper between logrus entry and gRPC logger interface.
type GRPC struct {
	*logrus.Entry
}

// V reports whether verbosity level is at least the requested verbose level.
func (v *GRPC) V(int) bool {
	// we don't need real implementation ATM
	return true
}

// Info logs a message at the Info level.
// Override InfoXXX methods with TraceXXX to keep gRPC and logrus levels in sync.
func (v *GRPC) Info(args ...any) { v.Trace(args...) }

// Infoln logs a message at the Info level.
func (v *GRPC) Infoln(args ...any) { v.Traceln(args...) }

// Infof logs a formatted message at the Info level.
func (v *GRPC) Infof(format string, args ...any) { v.Tracef(format, args...) }

// check interfaces.
var (
	_ grpclog.LoggerV2 = (*GRPC)(nil)
)
