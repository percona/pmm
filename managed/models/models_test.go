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

package models

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/percona/pmm/managed/utils/tests"
)

func TestLabels(t *testing.T) {
	t.Parallel()

	t.Run("Normal", func(t *testing.T) {
		t.Parallel()

		var b []byte
		err := setLabels(map[string]string{"_1foo": "bar", "baz": "  "}, &b)
		require.NoError(t, err)
		assert.JSONEq(t, `{"_1foo":"bar","baz":""}`, string(b))
		m, err := getLabels(b)
		require.NoError(t, err)
		assert.Equal(t, map[string]string{"_1foo": "bar", "baz": ""}, m)
	})

	t.Run("Empty", func(t *testing.T) {
		t.Parallel()

		var b []byte
		err := setLabels(make(map[string]string), &b)
		require.NoError(t, err)
		assert.Nil(t, b)
		m, err := getLabels(b)
		require.NoError(t, err)
		assert.Nil(t, m)
	})

	t.Run("Invalid", func(t *testing.T) {
		t.Parallel()

		var b []byte
		err := setLabels(map[string]string{"1": "bar"}, &b)
		tests.AssertGRPCError(t, status.New(codes.InvalidArgument, `Invalid label name "1".`), err)
	})

	t.Run("Reserved", func(t *testing.T) {
		t.Parallel()

		var b []byte
		err := setLabels(map[string]string{"__1": "bar"}, &b)
		tests.AssertGRPCError(t, status.New(codes.InvalidArgument, `Invalid label name "__1".`), err)
	})
}

// The two outcomes are not symmetric: an entry that yields no name stops the whole sweep, while one
// that yields a name is trusted as naming a live replica. Reading a name out of an entry that carries
// none would turn "keep every Node" into "remove every Node this entry didn't name".
func TestHAPeerNodeName(t *testing.T) {
	for _, tc := range []struct {
		peer string
		name string
		ok   bool
	}{
		{peer: "pmm-ha-0.monitoring-service.pmm.svc.cluster.local", name: "pmm-ha-0", ok: true}, // what the chart renders
		{peer: "pmm-ha-0.pmm-ha:9761", name: "pmm-ha-0", ok: true},
		{peer: " pmm-ha-1.pmm-ha.pmm.svc.cluster.local ", name: "pmm-ha-1", ok: true}, // trimmed
		{peer: "pmm-ha-2:9761", name: "pmm-ha-2", ok: true},                           // a dotless host with a port
		{peer: "pmm-ha-2", name: "pmm-ha-2", ok: true},
		{peer: "10.244.1.7"}, // bare IPv4, with and without a port
		{peer: "10.244.1.7:9761"},
		{peer: "2001:db8::7"}, // IPv6, unbracketed and bracketed
		{peer: "[2001:db8::7]:9761"},
		{peer: "[2001:db8::7]"},
		{peer: "pmm-ha-2/10.0.0.2"}, // memberlist's "name/address" form
		{peer: "pmm-ha-2/[2001:db8::7]:9761"},
		{peer: ":9761"},
		{peer: ""},
		{peer: "   "},
	} {
		t.Run(tc.peer, func(t *testing.T) {
			name, ok := haPeerNodeName(tc.peer)
			assert.Equal(t, tc.ok, ok)
			assert.Equal(t, tc.name, name)
		})
	}
}
