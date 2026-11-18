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
	"encoding/json"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testInput is the input of the tool that tests register through addTool.
type testInput struct {
	Panic bool   `json:"panic,omitempty"`
	Fail  string `json:"fail,omitempty"`
	Wait  bool   `json:"wait,omitempty"`
}

// testResult is the result of the tool that tests register through addTool.
type testResult struct {
	OK bool `json:"ok"`
}

func (testResult) render() string {
	return "ok"
}

// testTool panics or fails with the code in fail when asked to, and succeeds otherwise.
func testTool(_ context.Context, _ *mcp.CallToolRequest, in testInput) (testResult, error) {
	if in.Panic {
		panic("panic value")
	}
	if in.Fail != "" {
		return testResult{}, newToolError(errorCode(in.Fail), "failed")
	}
	return testResult{OK: true}, nil
}

// serveTestTool registers fn as pmm_test through addTool of a service built from params, on a server of its own, and
// returns a client session to it and the hook that captures the service's log at Trace.
func serveTestTool(t *testing.T, params Params, fn toolFunc[testInput, testResult]) (*mcp.ClientSession, *test.Hook) {
	t.Helper()

	s, err := New(params)
	require.NoError(t, err)
	logger, hook := test.NewNullLogger()
	logger.SetLevel(logrus.TraceLevel)
	s.l = logrus.NewEntry(logger).WithField("component", "mcp")

	server := mcp.NewServer(&mcp.Implementation{Name: "test", Version: "0"}, nil)
	addTool(s, server, &mcp.Tool{Name: "pmm_test"}, fn)

	clientTransport, serverTransport := mcp.NewInMemoryTransports()
	serverSession, err := server.Connect(t.Context(), serverTransport, nil)
	require.NoError(t, err)
	t.Cleanup(func() { _ = serverSession.Close() })
	client := mcp.NewClient(&mcp.Implementation{Name: "test", Version: "0"}, nil)
	session, err := client.Connect(t.Context(), clientTransport, nil)
	require.NoError(t, err)
	t.Cleanup(func() { _ = session.Close() })

	return session, hook
}

func TestFailureLogLevel(t *testing.T) {
	t.Parallel()

	session, hook := serveTestTool(t, Params{API: &mockPmmAPI{}}, testTool)
	for code, level := range map[errorCode]logrus.Level{
		codeInvalidInput:           logrus.DebugLevel,
		codeNotFound:               logrus.DebugLevel,
		codeRateLimited:            logrus.DebugLevel,
		codeUnauthorized:           logrus.DebugLevel,
		codeTimeout:                logrus.WarnLevel,
		codeNoQuerySource:          logrus.WarnLevel,
		codeInsufficientPrivileges: logrus.WarnLevel,
		codeAgentUnreachable:       logrus.WarnLevel,
		codePMMUnavailable:         logrus.WarnLevel,
	} {
		hook.Reset()
		res := callTool(t, session, "pmm_test", map[string]any{"fail": string(code)})
		require.True(t, res.IsError)

		var levels []logrus.Level
		for _, e := range hook.AllEntries() {
			if e.Message == "Tool call failed." {
				assert.Equal(t, string(code), e.Data["code"])
				levels = append(levels, e.Level)
			}
		}
		assert.Equal(t, []logrus.Level{level}, levels, string(code))
	}
}

func TestToolPanic(t *testing.T) {
	t.Parallel()

	session, hook := serveTestTool(t, Params{API: &mockPmmAPI{}}, testTool)

	res := callTool(t, session, "pmm_test", map[string]any{"panic": true})
	require.True(t, res.IsError)
	text := res.Content[0].(*mcp.TextContent).Text
	assert.True(t, strings.HasPrefix(text, "error: pmm_unavailable\n"), text)
	assert.NotContains(t, text, "panic value")
	assert.Equal(t, map[string]any{"code": "pmm_unavailable", "retryable": true}, res.Meta["com.percona/error"])

	var logged *logrus.Entry
	for _, e := range hook.AllEntries() {
		if e.Level == logrus.ErrorLevel {
			logged = e
		}
	}
	require.NotNil(t, logged, "the panic must be logged at Error")
	assert.Equal(t, "pmm_test", logged.Data["tool"])
	assert.Equal(t, "panic value", logged.Data["panic"])
	assert.Contains(t, logged.Data["stack"], "testTool")

	res = callTool(t, session, "pmm_test", nil)
	assert.False(t, res.IsError, "a call after the panic must work")
}

func TestStructuredAndErrorResults(t *testing.T) {
	t.Parallel()

	t.Run("Success", func(t *testing.T) {
		t.Parallel()

		fake := newFakePMM(t, map[string]fixture{"GET /v1/server/version": {file: "version.json"}})
		session := connect(t, newTestService(t, fake))
		res := callTool(t, session, "pmm_version", nil)
		require.False(t, res.IsError)
		var v versionResult
		require.NoError(t, json.Unmarshal(mustJSON(t, res.StructuredContent), &v))
		require.Equal(t, versionResult{Version: "3.8.1", ManagedVersion: "3.8.1-269-gfad5f58"}, v)
		require.Equal(t, "PMM Server 3.8.1 (pmm-managed 3.8.1-269-gfad5f58)", res.Content[0].(*mcp.TextContent).Text)
		require.NotNil(t, listTools(t, session)["pmm_version"].OutputSchema)
	})

	t.Run("Error", func(t *testing.T) {
		t.Parallel()

		// No route for /v1/server/version.
		fake := newFakePMM(t, nil)
		session := connect(t, newTestService(t, fake))
		res := callTool(t, session, "pmm_version", nil)
		require.True(t, res.IsError)
		require.Nil(t, res.StructuredContent)
		require.Equal(t, map[string]any{"code": "not_found", "retryable": false}, res.Meta["com.percona/error"])
		require.True(t, strings.HasPrefix(res.Content[0].(*mcp.TextContent).Text, "error: not_found\n"))
	})
}

func TestRetryable(t *testing.T) {
	t.Parallel()

	all := []errorCode{
		codeNotFound, codeInvalidInput, codeUnauthorized, codeTimeout, codeNoQuerySource,
		codeInsufficientPrivileges, codeAgentUnreachable, codePMMUnavailable, codeRateLimited,
	}
	want := map[errorCode]bool{codeTimeout: true, codeAgentUnreachable: true, codePMMUnavailable: true, codeRateLimited: true}
	for _, code := range all {
		assert.Equal(t, want[code], (&toolError{code: code}).retryable(), string(code))
	}
}
