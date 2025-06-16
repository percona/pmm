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

// Package logger contains logging utilities.
package logger

import (
	"context"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

// key is unexported to prevent collisions - it is different from any other type in other packages.
type keyStruct struct{}

var key = keyStruct{}

// Get returns logrus entry for given context. Set must be called before this method is called.
func Get(ctx context.Context) *logrus.Entry {
	v := ctx.Value(key)
	if v == nil {
		panic("context logger not set")
	}
	return v.(*logrus.Entry) //nolint:forcetypeassert
}

// Set returns derived context with set logrus entry with given request ID.
func Set(ctx context.Context, requestID string) context.Context {
	return SetEntry(ctx, logrus.WithField("request", requestID))
}

// SetEntry returns derived context with set given logrus entry.
func SetEntry(ctx context.Context, l *logrus.Entry) context.Context {
	if ctx.Value(key) != nil {
		Get(ctx).Panicf("context logger already set")
		return nil
	}

	return context.WithValue(ctx, key, l)
}

// MakeRequestID returns a new request ID.
func MakeRequestID() string {
	// UUID version 1: first 8 characters are time-based and lexicography sorted,
	// which is a useful property there
	u, err := uuid.NewUUID()
	if err != nil {
		panic(err)
	}

	return u.String()
}
