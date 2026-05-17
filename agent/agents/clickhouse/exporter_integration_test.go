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

//go:build clickhouse_integration

// Integration tests for the packaged clickhouse_exporter binary against real
// servers. Unlike collector_integration_test.go — which exercises the in-process
// Collector — this test launches the actual exporter binary the way pmm-agent
// runs it, then scrapes its /metrics endpoint over HTTP.
//
// The binary path is provided via CLICKHOUSE_EXPORTER_BIN (set by
// testdata/run-matrix.sh after building the binary once). When that variable is
// unset the test is skipped, so a plain `go test -tags clickhouse_integration`
// without the driver script still passes.
//
// The endpoint matrix is the same one collector_integration_test.go uses; the
// matrixEndpoints() helper is defined there and shared because both files are in
// package clickhouse.

package clickhouse

import (
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// exporterStartTimeout bounds how long the test waits for a freshly launched
// exporter to start answering /metrics with HTTP 200.
const exporterStartTimeout = 30 * time.Second

// exporterTeardownTimeout bounds how long the test waits for the exporter
// process to exit after it receives a kill signal.
const exporterTeardownTimeout = 10 * time.Second

// freeLocalPort asks the kernel for an unused TCP port on the loopback
// interface and returns it. The listener is closed before returning, so there
// is a tiny race window — acceptable for a single-process integration test.
func freeLocalPort(t *testing.T) int {
	t.Helper()
	var lc net.ListenConfig
	l, err := lc.Listen(context.Background(), "tcp", "127.0.0.1:0")
	require.NoError(t, err)
	port := l.Addr().(*net.TCPAddr).Port
	require.NoError(t, l.Close())
	return port
}

// scrapeMetrics performs a single GET against the exporter's /metrics endpoint
// and returns the response body. The bool reports whether the request returned
// HTTP 200; callers poll on it while the exporter is still starting up.
func scrapeMetrics(ctx context.Context, url string) (string, bool) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", false
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", false
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", false
	}
	return string(body), resp.StatusCode == http.StatusOK
}

// TestClickHouseExporterMatrix launches the real clickhouse_exporter binary
// against every configured endpoint and asserts its /metrics output exposes the
// three native metric families plus a successful scrape. An unreachable
// endpoint is skipped so the matrix can be run one topology at a time.
func TestClickHouseExporterMatrix(t *testing.T) {
	binPath := strings.TrimSpace(os.Getenv("CLICKHOUSE_EXPORTER_BIN"))
	if binPath == "" {
		t.Skip("CLICKHOUSE_EXPORTER_BIN not set; the exporter binary path is required")
	}
	require.FileExists(t, binPath, "CLICKHOUSE_EXPORTER_BIN must point to a built exporter binary")

	endpoints := matrixEndpoints()
	require.NotEmpty(t, endpoints)

	for name, dsn := range endpoints {
		t.Run(name, func(t *testing.T) {
			// Fail fast on an unreachable server before launching the binary,
			// so a missing endpoint is skipped rather than reported as a crash.
			c, err := NewCollector(dsn)
			if err != nil {
				t.Skipf("endpoint %q unreachable, skipping: %v", name, err)
			}
			require.NoError(t, c.Close())

			port := freeLocalPort(t)
			listenAddr := net.JoinHostPort("127.0.0.1", strconv.Itoa(port))
			metricsURL := "http://" + listenAddr + "/metrics"

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			cmd := exec.CommandContext(ctx, binPath, //nolint:gosec // binPath comes from the test harness
				"--clickhouse.dsn="+dsn,
				"--web.listen-address="+listenAddr,
				"--web.telemetry-path=/metrics",
			)
			cmd.Stdout = os.Stderr
			cmd.Stderr = os.Stderr
			require.NoError(t, cmd.Start(), "the exporter binary must start")

			// waitErr receives the process exit status once Wait returns; it is
			// read during teardown to assert a clean shutdown.
			waitErr := make(chan error, 1)
			go func() { waitErr <- cmd.Wait() }()

			// Poll /metrics until the exporter answers 200 or the timeout fires.
			deadline := time.Now().Add(exporterStartTimeout)
			var body string
			var ok bool
			for time.Now().Before(deadline) {
				scrapeCtx, scrapeCancel := context.WithTimeout(ctx, 3*time.Second)
				body, ok = scrapeMetrics(scrapeCtx, metricsURL)
				scrapeCancel()
				if ok {
					break
				}
				select {
				case err := <-waitErr:
					t.Fatalf("exporter exited before serving metrics: %v", err)
				case <-time.After(250 * time.Millisecond):
				}
			}
			require.True(t, ok, "the exporter must answer /metrics with HTTP 200 within %s", exporterStartTimeout)

			// Every native metric family must be present, plus a successful
			// scrape — exactly what a Prometheus server would see.
			assert.Contains(t, body, prefixMetrics, "system.metrics family must be exposed")
			assert.Contains(t, body, prefixProfileEvents, "system.events family must be exposed")
			assert.Contains(t, body, prefixAsyncMetrics, "system.asynchronous_metrics family must be exposed")
			assert.Contains(t, body, "clickhouse_exporter_last_scrape_success 1",
				"the last scrape against a healthy server must succeed")

			// Stop the exporter and assert a clean teardown: the process must
			// exit promptly once the context is canceled.
			cancel()
			select {
			case err := <-waitErr:
				// A context-canceled process exits non-zero (killed); that is
				// the expected teardown path, so only an unexpected error fails.
				if err != nil && !isSignalKill(err) {
					t.Fatalf("exporter did not shut down cleanly: %v", err)
				}
			case <-time.After(exporterTeardownTimeout):
				t.Fatalf("exporter did not exit within %s after shutdown", exporterTeardownTimeout)
			}
		})
	}
}

// TestClickHouseExporterVersion verifies the --version banner: pmm-agent's
// version probe matches the "clickhouse_exporter, version" prefix, so the
// banner must keep that exact shape.
func TestClickHouseExporterVersion(t *testing.T) {
	binPath := strings.TrimSpace(os.Getenv("CLICKHOUSE_EXPORTER_BIN"))
	if binPath == "" {
		t.Skip("CLICKHOUSE_EXPORTER_BIN not set; the exporter binary path is required")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	out, err := exec.CommandContext(ctx, binPath, "--version").CombinedOutput() //nolint:gosec // binPath from harness
	require.NoError(t, err, "clickhouse_exporter --version must exit 0")
	assert.Contains(t, string(out), "clickhouse_exporter, version",
		"the version banner must keep the prefix pmm-agent's probe matches")
}

// isSignalKill reports whether err is the exit status of a process terminated
// by a signal (the expected outcome of context cancellation), as opposed to a
// genuine failure exit code.
func isSignalKill(err error) bool {
	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) {
		return false
	}
	// A signal-killed process has no clean exit code; ProcessState reports it
	// as not exited normally.
	return !exitErr.Exited()
}
