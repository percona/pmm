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
)

func TestNewClickHouseParams(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		p, err := NewClickHouseParams("127.0.0.1:9000", "pmm", "default", "clickhouse")
		require.NoError(t, err)
		assert.Equal(t, "tcp://default:clickhouse@127.0.0.1:9000/pmm", p.URL().String())
	})

	t.Run("valid empty password", func(t *testing.T) {
		_, err := NewClickHouseParams("127.0.0.1:9000", "pmm", "default", "")
		require.NoError(t, err)
	})

	errCases := []struct {
		name       string
		addr       string
		dbName     string
		dbUsername string
		dbPassword string
		wantErrSub string
	}{
		{"empty addr", "", "pmm", "default", "clickhouse", "addr is required"},
		{"missing port", "127.0.0.1", "pmm", "default", "clickhouse", "invalid addr"},
		{"empty host", ":9000", "pmm", "default", "clickhouse", "empty host"},
		{"non numeric port", "localhost:abc", "pmm", "default", "clickhouse", "invalid port"},
		{"port out of range", "localhost:99999", "pmm", "default", "clickhouse", "invalid port"},
		{"empty db name", "127.0.0.1:9000", "", "default", "clickhouse", "database name is required"},
		{"empty username", "127.0.0.1:9000", "pmm", "", "clickhouse", "username is required"},
	}
	for _, tc := range errCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := NewClickHouseParams(tc.addr, tc.dbName, tc.dbUsername, tc.dbPassword)
			require.Error(t, err)
			assert.ErrorContains(t, err, tc.wantErrSub)
		})
	}
}

func TestCHParamsExternalClickHouse(t *testing.T) {
	cases := []struct {
		name string
		addr string
		want bool
	}{
		{"loopback", "127.0.0.1:9000", false},
		{"localhost", "localhost:9000", false},
		{"external host", "ch-01.test.net:9000", true},
		{"wildcard", "0.0.0.0:9000", true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			p, err := NewClickHouseParams(tc.addr, "pmm", "default", "clickhouse")
			require.NoError(t, err)
			assert.Equal(t, tc.want, p.ExternalClickHouse())
		})
	}
}

func TestCHParamsURL(t *testing.T) {
	t.Run("simple", func(t *testing.T) {
		p, err := NewClickHouseParams("127.0.0.1:9000", "pmm", "default", "clickhouse")
		require.NoError(t, err)
		assert.Equal(t, "tcp://default:clickhouse@127.0.0.1:9000/pmm", p.URL().String())
	})

	t.Run("password with special chars", func(t *testing.T) {
		p, err := NewClickHouseParams("127.0.0.1:9000", "pmm", "default", "p@ss/word")
		require.NoError(t, err)
		assert.Equal(t, "tcp://default:p%40ss%2Fword@127.0.0.1:9000/pmm", p.URL().String())
	})
}
