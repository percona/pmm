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

package ports

import (
	"errors"
	"fmt"
	"net"
	"sync"
)

var (
	errNoFreePort      = errors.New("ports registry: no free port")
	errPortBusy        = errors.New("ports registry: port busy")
	errNotReservedPort = errors.New("ports registry: not reserved port")
)

// Registry keeps track of reserved ports.
type Registry struct {
	lock     sync.Mutex
	min, max uint16
	m        map[uint16]struct{}
}

func NewRegistry(min, max uint16, reserved []uint16) *Registry {
	if min > max {
		panic("min > max")
	}

	m := make(map[uint16]struct{})
	for _, p := range reserved {
		m[p] = struct{}{}
	}

	return &Registry{
		min: min, max: max,
		m: m,
	}
}

func (r *Registry) Reserve() (uint16, error) {
	r.lock.Lock()
	defer r.lock.Unlock()

	for port := r.min; port <= r.max; port++ {
		if _, ok := r.m[port]; ok {
			continue
		}

		l, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
		if l != nil {
			l.Close()
		}
		if err != nil {
			continue
		}

		r.m[port] = struct{}{}
		return port, nil
	}

	return 0, errNoFreePort
}

func (r *Registry) Release(port uint16) error {
	r.lock.Lock()
	defer r.lock.Unlock()

	if _, ok := r.m[port]; !ok {
		return errNotReservedPort
	}

	l, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
	if l != nil {
		l.Close()
	}
	if err != nil {
		return errPortBusy
	}

	delete(r.m, port)
	return nil
}
