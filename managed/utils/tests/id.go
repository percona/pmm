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

package tests

import (
	"encoding/binary"
	"fmt"
	"io"
	"sync/atomic"
	"testing"

	"github.com/google/uuid"
)

// IDReader is used in tests for ID/UUID generation.
type IDReader struct {
	lastID uint64
}

// Read returns non-random data for ID/UUID generation.
func (t *IDReader) Read(b []byte) (int, error) {
	if len(b) != 16 {
		panic(fmt.Errorf("unexpected read of length %d", b))
	}

	for i := range b {
		b[i] = 0
	}
	id := atomic.AddUint64(&t.lastID, 1)
	binary.BigEndian.PutUint64(b[8:], id)
	return len(b), nil
}

// SetTestIDReader sets IDReader for duration of the test.
func SetTestIDReader(t *testing.T) {
	t.Helper()

	uuid.SetRand(&IDReader{})
	t.Cleanup(func() { uuid.SetRand(nil) })
}

// check interfaces.
var (
	_ io.Reader = (*IDReader)(nil)
)
