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
	return context.WithValue(ctx, key, l)
}
