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
	t.Setenv(EnvDSN, "")
	t.Setenv(EnvListenAddress, "")

	cfg := LoadConfig()

	assert.Equal(t, DefaultDSN, cfg.DSN)
	assert.Equal(t, DefaultListenAddress, cfg.ListenAddress)
	assert.Equal(t, DefaultTelemetryPath, cfg.TelemetryPath)
}

func TestLoadConfigFromEnv(t *testing.T) {
	t.Setenv(EnvDSN, "clickhouse://admin:secret@ch-host:9000/metrics")
	t.Setenv(EnvListenAddress, "0.0.0.0:9116")

	cfg := LoadConfig()

	assert.Equal(t, "clickhouse://admin:secret@ch-host:9000/metrics", cfg.DSN)
	assert.Equal(t, "0.0.0.0:9116", cfg.ListenAddress)
	assert.Equal(t, DefaultTelemetryPath, cfg.TelemetryPath)
}
