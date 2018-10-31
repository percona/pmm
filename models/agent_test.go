// pmm-managed
// Copyright (C) 2017 Percona LLC
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

	"github.com/AlekSi/pointer"
	"github.com/stretchr/testify/assert"
)

func TestDSN(t *testing.T) {
	t.Run("PostgresExporter", func(t *testing.T) {
		s := &PostgreSQLService{
			Address: pointer.ToString("hostname"),
			Port:    pointer.ToUint16(12345),
		}
		a := &PostgresExporter{
			ServiceUsername: pointer.ToString("username@1!"),
			ServicePassword: pointer.ToString("password@1!"),
		}
		expected := "postgres://username%401%21:password%401%21@hostname:12345/postgres?connect_timeout=5&sslmode=disable"
		assert.Equal(t, expected, a.DSN(s))
	})
}
