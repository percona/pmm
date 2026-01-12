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

// Package logger provides common logging utilities for all SaaS components.
package logger

import (
	"context"

	"go.uber.org/zap"
)

// key is unexported to prevent collisions - it is different from any other type in other packages
//
//nolint:gochecknoglobals
var key = struct{}{}

// GetLoggerFromContext returns logger from given context produced by GetContextWithLogger.
func GetLoggerFromContext(ctx context.Context) *zap.Logger {
	v := ctx.Value(key)
	if v == nil {
		l := zap.L()
		l.DPanic("context logger not set")

		return l
	}

	return v.(*zap.Logger) //nolint: forcetypeassert
}

// GetContextWithLogger returns derived context with given logger set.
// If logger is already present, it will be shadowed.
func GetContextWithLogger(ctx context.Context, l *zap.Logger) context.Context {
	return context.WithValue(ctx, key, l) //nolint:staticcheck
}
