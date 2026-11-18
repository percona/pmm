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

package mcp

import (
	"context"
	"maps"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// gathered returns the value of the series of metric name with exactly labels in reg: a counter's or a gauge's value,
// or a histogram's sample count. A series that reg doesn't hold is 0.
func gathered(t *testing.T, reg *prometheus.Registry, name string, labels map[string]string) float64 {
	t.Helper()

	families, err := reg.Gather()
	require.NoError(t, err)
	for _, family := range families {
		if family.GetName() != name {
			continue
		}
		for _, m := range family.GetMetric() {
			got := make(map[string]string, len(m.GetLabel()))
			for _, pair := range m.GetLabel() {
				got[pair.GetName()] = pair.GetValue()
			}
			if !maps.Equal(got, labels) {
				continue
			}
			switch {
			case m.GetCounter() != nil:
				return m.GetCounter().GetValue()
			case m.GetGauge() != nil:
				return m.GetGauge().GetValue()
			case m.GetHistogram() != nil:
				return float64(m.GetHistogram().GetSampleCount())
			}
		}
	}
	return 0
}

func TestToolCallMetrics(t *testing.T) {
	t.Parallel()

	reg := prometheus.NewRegistry()
	started := make(chan struct{})
	release := make(chan struct{})
	session, _ := serveTestTool(t, Params{API: &mockPmmAPI{}, Registerer: reg},
		func(ctx context.Context, req *mcp.CallToolRequest, in testInput) (testResult, error) {
			if in.Wait {
				close(started)
				<-release
			}
			return testTool(ctx, req, in)
		})

	tool := map[string]string{"tool": "pmm_test"}
	calls := func(code string) float64 {
		return gathered(t, reg, "pmm_managed_mcp_tool_calls_total", map[string]string{"tool": "pmm_test", "code": code})
	}

	// Around one call.
	done := make(chan *mcp.CallToolResult)
	go func() {
		res, err := session.CallTool(t.Context(), &mcp.CallToolParams{Name: "pmm_test", Arguments: map[string]any{"wait": true}})
		assert.NoError(t, err)
		done <- res
	}()

	<-started
	assert.InDelta(t, 1, gathered(t, reg, "pmm_managed_mcp_tool_calls_in_flight", tool), 0)
	assert.Zero(t, gathered(t, reg, "pmm_managed_mcp_tool_duration_seconds", tool))
	assert.Zero(t, calls("ok"))

	close(release)
	res := <-done
	require.NotNil(t, res)
	assert.False(t, res.IsError)
	assert.Zero(t, gathered(t, reg, "pmm_managed_mcp_tool_calls_in_flight", tool))
	assert.InDelta(t, 1, gathered(t, reg, "pmm_managed_mcp_tool_duration_seconds", tool), 0)
	assert.InDelta(t, 1, calls("ok"), 0)

	// After each error code, a success and a panic.
	codes := []errorCode{
		codeNotFound, codeInvalidInput, codeUnauthorized, codeTimeout, codeNoQuerySource,
		codeInsufficientPrivileges, codeAgentUnreachable, codePMMUnavailable, codeRateLimited,
	}
	for _, code := range codes {
		res = callTool(t, session, "pmm_test", map[string]any{"fail": string(code)})
		require.True(t, res.IsError)
		assert.InDelta(t, 1, calls(string(code)), 0, code)
	}

	res = callTool(t, session, "pmm_test", nil)
	require.False(t, res.IsError)
	assert.InDelta(t, 2, calls("ok"), 0)

	res = callTool(t, session, "pmm_test", map[string]any{"panic": true})
	require.True(t, res.IsError)
	assert.InDelta(t, 2, calls(string(codePMMUnavailable)), 0, "a panic counts as pmm_unavailable")

	assert.InDelta(t, float64(len(codes)+3), gathered(t, reg, "pmm_managed_mcp_tool_duration_seconds", tool), 0)
	assert.Zero(t, gathered(t, reg, "pmm_managed_mcp_tool_calls_in_flight", tool))
}

func TestToolCallMetricsRegistration(t *testing.T) {
	t.Parallel()

	reg := prometheus.NewRegistry()
	_, err := New(Params{API: &mockPmmAPI{}, Registerer: reg})
	require.NoError(t, err)

	_, err = New(Params{API: &mockPmmAPI{}, Registerer: reg})
	require.Error(t, err, "a second service must not register the same metrics on the same registerer")

	_, err = New(Params{API: &mockPmmAPI{}})
	require.NoError(t, err, "without a registerer the metrics are not registered anywhere")
}
