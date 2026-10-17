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
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/percona/pmm/managed/utils/secrets"
)

// TestExecEnvHelper is run by TestExecEnv in a child process, as exec-env replaces the process. Run with
// PMM_TEST_EXEC_ENV=exec, it runs itself through runExecEnv, which sets the rotated variables of the file at
// PMM_TEST_ROTATED_ENV; run with PMM_TEST_EXEC_ENV=print, it prints the variable it got.
func TestExecEnvHelper(t *testing.T) { //nolint:paralleltest
	switch os.Getenv("PMM_TEST_EXEC_ENV") {
	case "exec":
		require.NoError(t, os.Setenv("PMM_TEST_EXEC_ENV", "print")) //nolint:usetesting // for the process exec-env runs
		err := runExecEnv([]string{os.Args[0], "-test.run=^TestExecEnvHelper$"}, os.Getenv("PMM_TEST_ROTATED_ENV"))
		require.NoError(t, err, "only returns if it could not run the program")
	case "print":
		_, _ = os.Stdout.WriteString("value=" + os.Getenv("GF_SMTP_PASSWORD") + "\n")
		os.Exit(0)
	default:
		t.Skip("run by TestExecEnv")
	}
}

func TestExecEnv(t *testing.T) {
	t.Parallel()

	rotated := filepath.Join(t.TempDir(), "rotated-env.json")
	run := func(t *testing.T) string {
		t.Helper()
		cmd := exec.CommandContext(t.Context(), os.Args[0], "-test.run=^TestExecEnvHelper$")
		cmd.Env = append(os.Environ(), "PMM_TEST_EXEC_ENV=exec", "PMM_TEST_ROTATED_ENV="+rotated, "GF_SMTP_PASSWORD=started-with")
		out, err := cmd.Output()
		require.NoError(t, err)
		return string(out)
	}

	// Nothing rotated: the value it started with.
	assert.Contains(t, run(t), "value=started-with\n")

	require.NoError(t, secrets.WriteRotatedEnv(rotated, map[string]string{"GF_SMTP_PASSWORD": "rotated"}))
	assert.Contains(t, run(t), "value=rotated\n")

	require.Error(t, runExecEnv(nil, rotated))
	require.Error(t, runExecEnv([]string{"/nonexistent/program"}, rotated))
}
