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

package clickhouse

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLoadConfigDefaults(t *testing.T) {
	t.Setenv("CLICKHOUSE_DSN", "")
	t.Setenv("CLICKHOUSE_SCRAPE_PORT", "")

	cfg := LoadConfig()

	assert.Equal(t, "tcp://localhost:9000?username=default&password=&database=default", cfg.DSN)
	assert.Equal(t, "9100", cfg.ScrapePort)
}

func TestLoadConfigFromEnv(t *testing.T) {
	t.Setenv("CLICKHOUSE_DSN", "tcp://ch-host:9000?username=admin&password=secret&database=metrics")
	t.Setenv("CLICKHOUSE_SCRAPE_PORT", "9101")

	cfg := LoadConfig()

	assert.Equal(t, "tcp://ch-host:9000?username=admin&password=secret&database=metrics", cfg.DSN)
	assert.Equal(t, "9101", cfg.ScrapePort)
}
