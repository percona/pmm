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

package dbwatcher

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDBLogWatcherTailsAndShips(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "error.log")
	require.NoError(t, os.WriteFile(logPath, []byte("preexisting line\n"), 0o600))

	w := New(&Params{
		AgentID:     "agent-1",
		ServiceID:   "service-1",
		ServiceName: "mysql-svc",
		DBSystem:    "mysql",
		Files:       []WatchedFile{{Path: logPath, Type: "error"}},
	}, logrus.WithField("test", t.Name()))

	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	defer cancel()

	go w.Run(ctx)

	// The reader seeks to the end on open, so write a new line after starting.
	f, err := os.OpenFile(logPath, os.O_APPEND|os.O_WRONLY, 0o600) //nolint:gosec
	require.NoError(t, err)
	// Give the watcher a moment to open and seek before appending.
	time.Sleep(500 * time.Millisecond)
	_, err = f.WriteString("2026-06-03T10:00:00Z 0 [ERROR] [MY-010119] [Server] Aborting\n")
	require.NoError(t, err)
	require.NoError(t, f.Close())

	for {
		select {
		case <-ctx.Done():
			t.Fatal("timed out waiting for a shipped log record")
		case change := <-w.Changes():
			if len(change.LogShipRequests) == 0 {
				continue
			}
			req := change.LogShipRequests[0]
			assert.Equal(t, "mysql-svc", req.ServiceName)
			assert.Equal(t, "mysql", req.ResourceAttributes["db.system"])
			assert.Equal(t, "client", req.ResourceAttributes["pmm.source"])
			require.Len(t, req.Records, 1)
			assert.Contains(t, req.Records[0].Body, "[ERROR]")
			assert.Equal(t, "error", req.Records[0].Attributes["pmm.log_type"])
			return
		}
	}
}

func TestDBLogWatcherAllowlistRejectsOutsidePaths(t *testing.T) {
	w := New(&Params{
		Files:       []WatchedFile{{Path: "/etc/shadow", Type: "error"}},
		AllowedDirs: []string{t.TempDir()},
	}, logrus.WithField("test", t.Name()))

	_, err := w.validatePath("/etc/shadow")
	require.Error(t, err)
}
