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
	"fmt"
	"net"
	"sync"
)

var (
	errNoFreePort      = fmt.Errorf("no free port")
	errPortBusy        = fmt.Errorf("port busy")
	errPortNotReserved = fmt.Errorf("port not reserved")
)

// portsRegistry keeps track of reserved ports.
type portsRegistry struct {
	m        sync.Mutex
	min      uint16
	max      uint16
	last     uint16
	reserved map[uint16]struct{}
}

func newPortsRegistry(min, max uint16, reserved []uint16) *portsRegistry {
	if min > max {
		panic(fmt.Sprintf("min port (%d) > max port (%d)", min, max))
	}

	r := &portsRegistry{
		min:      min,
		max:      max,
		last:     min - 1,
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
func (r *portsRegistry) Reserve() (uint16, error) {
	r.m.Lock()
	defer r.m.Unlock()

	size := r.max - r.min + 1
	for i := uint16(1); i <= size; i++ {
		port := r.min + (r.last-r.min+i)%size
		if _, ok := r.reserved[port]; ok {
			continue
		}

		l, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
		if l != nil {
			_ = l.Close()
		}
		if err != nil {
			continue
		}

		r.reserved[port] = struct{}{}
		r.last = port
		return port, nil
	}

	return 0, errNoFreePort
}

// Release releases port.
func (r *portsRegistry) Release(port uint16) error {
	r.m.Lock()
	defer r.m.Unlock()

	if _, ok := r.reserved[port]; !ok {
		return errPortNotReserved
	}

	l, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
	if l != nil {
		_ = l.Close()
	}
	if err != nil {
		return errPortBusy
	}

	delete(r.reserved, port)
	return nil
}
