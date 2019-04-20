// pmm-agent
// Copyright (C) 2018 Percona LLC
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

package client

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/percona/pmm-agent/config"
)

func TestClient(t *testing.T) {
	t.Run("NoAddress", func(t *testing.T) {
		t.Parallel()
		ctx, cancel := context.WithCancel(context.Background())

		cfg := &config.Config{}
		client := New(cfg, nil)
		cancel()
		err := client.Run(ctx)
		assert.Equal(t, "missing PMM Server address: context canceled", err.Error())
	})

	t.Run("NoAgentID", func(t *testing.T) {
		t.Parallel()
		ctx, cancel := context.WithCancel(context.Background())

		cfg := &config.Config{
			Server: config.Server{
				Address: "127.0.0.1:65000",
			},
		}
		client := New(cfg, nil)
		cancel()
		err := client.Run(ctx)
		assert.Equal(t, "missing Agent ID: context canceled", err.Error())
	})

	t.Run("FailedToConnect", func(t *testing.T) {
		t.Parallel()
		ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
		defer cancel()

		cfg := &config.Config{
			ID: "agent_id",
			Server: config.Server{
				Address: "127.0.0.1:65000",
			},
		}
		client := New(cfg, nil)
		err := client.Run(ctx)
		assert.Equal(t, "failed to connect: context deadline exceeded", err.Error())
	})
}
