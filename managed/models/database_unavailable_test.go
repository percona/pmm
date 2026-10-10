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

package models_test

import (
	"encoding/binary"
	"io"
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/percona/pmm/managed/models"
)

// fakePostgres accepts connections and answers every startup message with an ErrorResponse
// made of the given fields, or closes the connection if there are none.
func fakePostgres(t *testing.T, errFields ...[2]string) string {
	t.Helper()

	l, err := new(net.ListenConfig).Listen(t.Context(), "tcp", "127.0.0.1:0")
	require.NoError(t, err)
	t.Cleanup(func() { _ = l.Close() })

	var fields []byte
	for _, f := range errFields {
		fields = append(fields, f[0][0])
		fields = append(fields, f[1]...)
		fields = append(fields, 0)
	}
	fields = append(fields, 0)
	resp := binary.BigEndian.AppendUint32([]byte{'E'}, uint32(len(fields)+4))
	resp = append(resp, fields...)

	go func() {
		for {
			conn, err := l.Accept()
			if err != nil {
				return
			}
			var size uint32
			if binary.Read(conn, binary.BigEndian, &size) == nil {
				_, _ = io.CopyN(io.Discard, conn, int64(size)-4)
				if len(errFields) > 0 {
					_, _ = conn.Write(resp)
				}
			}
			_ = conn.Close()
		}
	}()

	return l.Addr().String()
}

func fatal(code, message string) [][2]string {
	return [][2]string{{"S", "FATAL"}, {"V", "FATAL"}, {"C", code}, {"M", message}}
}

func TestOpenDBUnavailable(t *testing.T) {
	t.Parallel()

	ping := func(t *testing.T, address string) error {
		t.Helper()

		db, err := models.OpenDB(models.SetupDBParams{Address: address, Username: "pmm-managed", Password: "pmm-managed"})
		require.NoError(t, err)
		t.Cleanup(func() { _ = db.Close() })
		return db.PingContext(t.Context())
	}

	t.Run("connection refused", func(t *testing.T) {
		t.Parallel()

		l, err := new(net.ListenConfig).Listen(t.Context(), "tcp", "127.0.0.1:0")
		require.NoError(t, err)
		address := l.Addr().String()
		require.NoError(t, l.Close())

		assert.ErrorIs(t, ping(t, address), models.ErrDatabaseUnavailable)
	})

	t.Run("database system is shutting down", func(t *testing.T) {
		t.Parallel()

		err := ping(t, fakePostgres(t, fatal("57P03", "the database system is shutting down")...))
		require.ErrorIs(t, err, models.ErrDatabaseUnavailable)
		assert.ErrorContains(t, err, "the database system is shutting down")
	})

	t.Run("database system is shutting down after startup began", func(t *testing.T) {
		t.Parallel()

		err := ping(t, fakePostgres(t, fatal("57P01", "terminating connection due to administrator command")...))
		require.ErrorIs(t, err, models.ErrDatabaseUnavailable)
	})

	t.Run("too many connections", func(t *testing.T) {
		t.Parallel()

		err := ping(t, fakePostgres(t, fatal("53300", "sorry, too many clients already")...))
		require.ErrorIs(t, err, models.ErrDatabaseUnavailable)
	})

	t.Run("connection closed during startup", func(t *testing.T) {
		t.Parallel()

		require.ErrorIs(t, ping(t, fakePostgres(t)), models.ErrDatabaseUnavailable)
	})

	t.Run("protocol violation is not retryable", func(t *testing.T) {
		t.Parallel()

		err := ping(t, fakePostgres(t, fatal("08P01", "invalid startup packet layout")...))
		require.Error(t, err)
		assert.NotErrorIs(t, err, models.ErrDatabaseUnavailable)
	})

	t.Run("error without SQLSTATE is not retryable", func(t *testing.T) {
		t.Parallel()

		err := ping(t, fakePostgres(t, [2]string{"S", "FATAL"}, [2]string{"M", "no code"}))
		require.Error(t, err)
		assert.NotErrorIs(t, err, models.ErrDatabaseUnavailable)
	})

	t.Run("authentication failure is not retryable", func(t *testing.T) {
		t.Parallel()

		err := ping(t, fakePostgres(t, fatal("28P01", "password authentication failed")...))
		require.Error(t, err)
		assert.NotErrorIs(t, err, models.ErrDatabaseUnavailable)
	})
}
