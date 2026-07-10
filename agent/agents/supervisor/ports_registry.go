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

package supervisor

import (
	"context"
	"errors"
	"fmt"
	"net"
	"sync"
)

var (
	errNoFreePort      = errors.New("no free port")
	errPortBusy        = errors.New("port busy")
	errPortNotReserved = errors.New("port not reserved")
)

// portsRegistry keeps track of reserved ports.
type portsRegistry struct {
	m        sync.Mutex
	min      uint16
	max      uint16
	last     uint16
	reserved map[uint16]struct{}
}

func newPortsRegistry(minPort, maxPort uint16, reserved []uint16) *portsRegistry {
	if minPort > maxPort {
		panic(fmt.Sprintf("min port (%d) > max port (%d)", minPort, maxPort))
	}

	r := &portsRegistry{
		min:      minPort,
		max:      maxPort,
		last:     minPort - 1,
		reserved: make(map[uint16]struct{}, len(reserved)),
	}
	for _, p := range reserved {
		r.reserved[p] = struct{}{}
	}

	return r
}

// Reserve reserves next free port.
// It tries to reuse ports as little as possible to avoid erroneous Prometheus scrapes
// to the different exporter type when Prometheus configuration is being reloaded.
func (r *portsRegistry) Reserve(ctx context.Context) (uint16, error) {
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	r.m.Lock()
	defer r.m.Unlock()
	size := r.max - r.min + 1
	for i := uint16(1); i <= size; i++ {
		port := r.min + (r.last-r.min+i)%size
		if _, ok := r.reserved[port]; ok {
			continue
		}

		lc := net.ListenConfig{}
		l, err := lc.Listen(ctx, "tcp", fmt.Sprintf("127.0.0.1:%d", port))
		if err != nil {
			if ctx.Err() != nil {
				return 0, ctx.Err()
			}
			continue
		}
		if l != nil {
			_ = l.Close()
		}

		r.reserved[port] = struct{}{}
		r.last = port
		return port, nil
	}

	return 0, errNoFreePort
}

// Release releases port.
func (r *portsRegistry) Release(ctx context.Context, port uint16) error {
	r.m.Lock()
	defer r.m.Unlock()

	if _, ok := r.reserved[port]; !ok {
		return errPortNotReserved
	}

	lc := net.ListenConfig{}
	l, err := lc.Listen(ctx, "tcp", fmt.Sprintf("127.0.0.1:%d", port))
	if err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		return errPortBusy
	}
	if l != nil {
		_ = l.Close()
	}

	delete(r.reserved, port)
	return nil
}
