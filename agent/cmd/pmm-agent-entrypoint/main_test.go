// Copyright (C) 2023 Percona LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupFakePmmAgent puts a fake pmm-agent on PATH that logs each start, then runs body.
func setupFakePmmAgent(t *testing.T, body string) string {
	t.Helper()
	dir := t.TempDir()
	startsFile := filepath.Join(dir, "starts")
	script := fmt.Sprintf("#!/bin/sh\necho \"$@\" >> %s\n%s\n", startsFile, body)
	err := os.WriteFile(filepath.Join(dir, "pmm-agent"), []byte(script), 0o755)
	require.NoError(t, err)
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	return startsFile
}

// failingPrerun returns a prerun script that fails the first failures runs, and the file counting its runs.
func failingPrerun(t *testing.T, failures int) (string, string) {
	t.Helper()
	attemptsFile := filepath.Join(t.TempDir(), "attempts")
	script := fmt.Sprintf("echo x >> %[1]s; [ $(wc -l < %[1]s) -gt %[2]d ] || exit 3", attemptsFile, failures)
	return script, attemptsFile
}

func countLines(t *testing.T, path string) int {
	t.Helper()
	b, err := os.ReadFile(path)
	require.NoError(t, err)
	return len(strings.Split(strings.TrimSpace(string(b)), "\n"))
}

// runsUntilSIGTERM keeps the fake pmm-agent running until it gets SIGTERM, then exits 0.
const runsUntilSIGTERM = "trap 'exit 0' TERM\nwhile :; do sleep 0.1; done"

func TestRunPrerun(t *testing.T) {
	l := logrus.WithField("test", t.Name())

	t.Run("SidecarRetriesUntilSuccess", func(t *testing.T) {
		startsFile := setupFakePmmAgent(t, runsUntilSIGTERM)
		script, attemptsFile := failingPrerun(t, 2)

		status := runPrerun(t.Context(), l, "", script, true, 0)

		assert.Equal(t, 0, status)
		assert.Equal(t, 3, countLines(t, attemptsFile))
		assert.Equal(t, 1, countLines(t, startsFile), "pmm-agent must keep running across retries")
	})

	t.Run("NoRetryReturnsFailure", func(t *testing.T) {
		setupFakePmmAgent(t, runsUntilSIGTERM)
		script, attemptsFile := failingPrerun(t, 2)

		status := runPrerun(t.Context(), l, "", script, false, 0)

		assert.Equal(t, 3, status)
		assert.Equal(t, 1, countLines(t, attemptsFile))
	})

	t.Run("PrerunFile", func(t *testing.T) {
		setupFakePmmAgent(t, runsUntilSIGTERM)
		script, attemptsFile := failingPrerun(t, 1)
		prerunFile := filepath.Join(t.TempDir(), "prerun.sh")
		err := os.WriteFile(prerunFile, []byte("#!/bin/sh\n"+script+"\n"), 0o755)
		require.NoError(t, err)

		status := runPrerun(t.Context(), l, prerunFile, "", true, 0)

		assert.Equal(t, 0, status)
		assert.Equal(t, 2, countLines(t, attemptsFile))
	})

	t.Run("StopsRetryingWhenPmmAgentExits", func(t *testing.T) {
		setupFakePmmAgent(t, "sleep 0.3; exit 2")
		script, attemptsFile := failingPrerun(t, 1000)

		status := runPrerun(t.Context(), l, "", script, true, 100*time.Millisecond)

		assert.Equal(t, 2, status)
		assert.Less(t, countLines(t, attemptsFile), 10)
	})
}
