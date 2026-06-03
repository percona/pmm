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

package supervisord

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMarshalOtelcolConfig(t *testing.T) {
	t.Setenv("PMM_CLICKHOUSE_ADDR", "127.0.0.1:9000")
	t.Setenv("PMM_CLICKHOUSE_DATABASE", "pmm")
	t.Setenv("PMM_CLICKHOUSE_USER", "default")
	t.Setenv("PMM_CLICKHOUSE_PASSWORD", "clickhouse")

	t.Run("renders config", func(t *testing.T) {
		cfg, err := marshalOtelcolConfig()
		require.NoError(t, err)
		s := string(cfg)
		assert.Contains(t, s, `endpoint: "tcp://127.0.0.1:9000?dial_timeout=10s&compress=lz4"`)
		assert.Contains(t, s, `database: "pmm"`)
		assert.Contains(t, s, "logs_table_name: logs")
		assert.Contains(t, s, "traces_table_name: traces")
		assert.Contains(t, s, "create_schema: false")
		assert.Contains(t, s, "/srv/logs/*.log")
		assert.Contains(t, s, "127.0.0.1:4317")
	})

	t.Run("bad addr", func(t *testing.T) {
		t.Setenv("PMM_CLICKHOUSE_ADDR", "no-port")
		_, err := marshalOtelcolConfig()
		require.Error(t, err)
	})
}
