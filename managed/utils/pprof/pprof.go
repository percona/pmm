// Copyright (C) 2024 Percona LLC
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

// Package pprof contains profiling functionality.
package pprof

import (
	"bytes"
	"context"
	"runtime"
	"runtime/pprof"
	"runtime/trace"
	"time"

	"github.com/pkg/errors"
)

// Profile responds with the pprof-formatted cpu profile.
// Profiling lasts for duration specified in seconds.
func Profile(ctx context.Context, duration time.Duration) ([]byte, error) {
	var profileBuf bytes.Buffer
	if err := pprof.StartCPUProfile(&profileBuf); err != nil {
		return nil, err
	}

	select {
	case <-time.After(duration):
		pprof.StopCPUProfile()
		return profileBuf.Bytes(), nil
	case <-ctx.Done():
		pprof.StopCPUProfile()
		return nil, errors.New("pprof.Profile was canceled")
	}
}

// Trace responds with the execution trace in binary form.
// Tracing lasts for duration specified in seconds.
func Trace(ctx context.Context, duration time.Duration) ([]byte, error) {
	var traceBuf bytes.Buffer
	if err := trace.Start(&traceBuf); err != nil {
		return nil, err
	}

	select {
	case <-time.After(duration):
		trace.Stop()
		return traceBuf.Bytes(), nil
	case <-ctx.Done():
		trace.Stop()
		return nil, errors.New("pprof.Trace was canceled")
	}
}

// Heap responds with the pprof-formatted profile named "heap". Listing the available profiles.
// You can specify the gc parameter to run gc before taking the heap sample.
func Heap(gc bool) ([]byte, error) {
	var heapBuf bytes.Buffer
	debug := 0
	profile := "heap"

	p := pprof.Lookup(profile)
	if p == nil {
		return nil, errors.Errorf("profile cannot be found: %s", profile)
	}

	if gc {
		runtime.GC()
	}

	err := p.WriteTo(&heapBuf, debug)
	if err != nil {
		return nil, err
	}

	return heapBuf.Bytes(), nil
}
