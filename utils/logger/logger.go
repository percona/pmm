// pmm-managed
// Copyright (C) 2017 Percona LLC
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published
// by the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program. If not, see <http://www.gnu.org/licenses/>.

// Package logger contains logging utilities.
package logger

import (
	"context"
	"fmt"
	"time"

	"github.com/sirupsen/logrus"
)

// key is unexported to prevent collisions - it is different from any other type in other packages
type key struct{}

// Get returns logrus entry for given context. Set must be called before this method is called.
func Get(ctx context.Context) *logrus.Entry {
	v := ctx.Value(key{})
	if v == nil {
		panic("context logger not set")
	}
	return v.(*logrus.Entry)
}

// Set returns derived context with set logrus entry with given request ID.
func Set(ctx context.Context, requestID string) (context.Context, *logrus.Entry) {
	if ctx.Value(key{}) != nil {
		Get(ctx).Panicf("context logger already set")
		return nil, nil
	}

	l := logrus.WithField("request", requestID)
	return context.WithValue(ctx, key{}, l), l
}

// MakeRequestID returns a new request ID.
func MakeRequestID() string {
	return fmt.Sprintf("%016x", time.Now().UnixNano())
}
