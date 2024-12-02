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

package main

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"testing"
	"time"

	"github.com/percona/saas/pkg/check"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	agentv1 "github.com/percona/pmm/api/agent/v1"
	"github.com/percona/pmm/managed/services/checks"
)

const (
	invalidStarlarkScriptStderr = "Error running starlark script: thread invalid starlark script: failed to execute function check_context: function check_context accepts no arguments (2 given)"

	// Possible errors:
	// fatal error: runtime: out of memory
	// fatal error: out of memory allocating heap arena metadatai.
	memoryConsumingScriptStderr = "out of memory"
)

var validQueryActionResult = []map[string]interface{}{
	{"Value": "5.7.30-33-log", "Variable_name": "version"},
	{"Value": "Percona Server (GPL), Release 33, Revision 6517692", "Variable_name": "version_comment"},
	{"Value": "x86_64", "Variable_name": "version_compile_machine"},
	{"Value": "Linux", "Variable_name": "version_compile_os"},
	{"Value": "-log", "Variable_name": "version_suffix"},
}

func TestStarlarkSandbox(t *testing.T) { //nolint:tparallel
	testCases := []struct {
		name         string
		script       string
		exitError    string
		stderr       string
		checkResults []check.Result
		exitCode     int
	}{
		{
			name:         "invalid starlark script",
			script:       "def check_context(): return []",
			exitError:    "exit status 1",
			stderr:       invalidStarlarkScriptStderr,
			checkResults: nil,
			exitCode:     1,
		}, {
			name:         "memory consuming starlark script",
			script:       "def check_context(rows, context): return [1] * (1 << 30-1)",
			exitError:    "exit status 2",
			stderr:       memoryConsumingScriptStderr,
			checkResults: nil,
			exitCode:     2,
		}, {
			name: "cpu consuming starlark script",
			script: `def check_context(rows, context):
							while True:
								pass`,
			exitError:    "signal: killed",
			stderr:       "",
			checkResults: nil,
			exitCode:     -1,
		}, {
			name: "valid starlark script",
			script: `def check_context(rows, context):
							results = []
							results.append({
								"summary": "Fake check",
								"description": "Fake check description",
								"severity": "warning",
							})
							return results`,
			exitError: "",
			stderr:    "",
			checkResults: []check.Result{
				{
					Summary:     "Fake check",
					Description: "Fake check description",
					Severity:    5,
					Labels:      nil,
				},
			},
			exitCode: 0,
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	t.Cleanup(cancel)
	// since we run the binary as a child process to test it we need to build it first.
	command := exec.CommandContext(ctx, "make", "-C", "../..", "release-starlark")
	command.Stdout = os.Stdout
	command.Stderr = os.Stderr
	err := command.Run()
	require.NoError(t, err)

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			result, err := agentv1.MarshalActionQueryDocsResult(validQueryActionResult)
			require.NoError(t, err)

			data := &checks.StarlarkScriptData{
				Version:        1,
				Name:           tc.name,
				Script:         tc.script,
				QueriesResults: []any{result},
			}

			releasePath, present := os.LookupEnv("PMM_RELEASE_PATH")
			if !present {
				releasePath = "./../../bin"
			}
			cmd := exec.Command(releasePath + "/pmm-managed-starlark") //nolint:gosec

			var stdin, stderr bytes.Buffer
			cmd.Stdin = &stdin
			cmd.Stderr = &stderr
			cmd.Env = []string{starlarkRecursionFlag + "=1"}

			encoder := json.NewEncoder(&stdin)
			err = encoder.Encode(data)
			require.NoError(t, err)

			actualStdout, err := cmd.Output()
			if err != nil {
				exiterr, ok := err.(*exec.ExitError) //nolint:errorlint
				require.True(t, ok)
				assert.Equal(t, tc.exitError, exiterr.Error())
				assert.Equal(t, tc.exitCode, exiterr.ExitCode())
			}

			if tc.checkResults != nil {
				var expectedStdout bytes.Buffer
				encoder := json.NewEncoder(&expectedStdout)
				err = encoder.Encode(tc.checkResults)
				require.NoError(t, err)
				assert.Equal(t, expectedStdout.String(), string(actualStdout))
			}

			stderrContent := stderr.String()
			assert.Contains(t, stderrContent, tc.stderr)

			// make sure that the limits were set
			assert.NotContains(t, stderrContent, cpuUsageWarning)
			assert.NotContains(t, stderrContent, memoryUsageWarning)
		})
	}
}
