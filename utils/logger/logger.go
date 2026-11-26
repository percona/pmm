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

// Package logger contains logging utilities.
package logger

import (
	"context"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

// key is unexported to prevent collisions - it is different from any other type in other packages.
type key struct{}

// Get returns logrus entry for given context. Set must be called before this method is called.
func Get(ctx context.Context) *logrus.Entry {
	v := ctx.Value(key{})
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
	if ctx.Value(key{}) != nil {
		Get(ctx).Panicf("context logger already set")
		return nil
	}

	return context.WithValue(ctx, key{}, l)
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
