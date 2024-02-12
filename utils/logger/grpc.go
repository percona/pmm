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

package logger

import (
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc/grpclog"
)

// GRPC is a compatibility wrapper between logrus entry and gRPC logger interface.
type GRPC struct {
	*logrus.Entry
}

// V reports whether verbosity level l is at least the requested verbose level.
func (v *GRPC) V(l int) bool { //nolint:revive
	// we don't need real implementation ATM
	return true
}

// Info logs a message at the Info level.
// override InfoXXX methods with TraceXXX to keep gRPC and logrus levels in sync
//
//nolint:stylecheck
func (v *GRPC) Info(args ...interface{}) { v.Trace(args...) }

// Infoln logs a message at the Info level.
func (v *GRPC) Infoln(args ...interface{}) { v.Traceln(args...) }

// Infof logs a formatted message at the Info level.
func (v *GRPC) Infof(format string, args ...interface{}) { v.Tracef(format, args...) }

// check interfaces.
var (
	_ grpclog.LoggerV2 = (*GRPC)(nil)
)
