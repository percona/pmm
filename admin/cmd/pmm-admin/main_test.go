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

package main

import (
	"bytes"
	"encoding/json"
	"os/exec"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPackages(t *testing.T) {
	cmd := exec.CommandContext(t.Context(), "pmm-admin", "-h")
	b, err := cmd.CombinedOutput()
	require.NoError(t, err, "%s", b)

	out := string(b)
	assert.NotContains(t, out, "httptest.serve", `pmm-admin should not import package "net/http/httptest"`)
	assert.NotContains(t, out, "test.run", `pmm-admin should not import package "testing"`)
}

func TestVersionPlain(t *testing.T) {
	cmd := exec.CommandContext(t.Context(), "pmm-admin", "--version")
	b, err := cmd.CombinedOutput()
	require.NoError(t, err, "%s", b)

	out := string(b)
	assert.Contains(t, out, `Version:`, `--version output is incorrect`)
}

func TestNoCommandPrintsUsage(t *testing.T) {
	// --server-url avoids the local pmm-agent lookup so the test does not depend
	// on a running agent; with no subcommand pmm-admin must print usage instead
	// of panicking (PMM-15242).
	cmd := exec.CommandContext(t.Context(), "pmm-admin", "--server-url=http://localhost/")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()

	var exitErr *exec.ExitError
	require.ErrorAs(t, err, &exitErr, "stdout=%q stderr=%q", stdout.String(), stderr.String())
	assert.Equal(t, 80, exitErr.ExitCode())
	assert.Contains(t, stdout.String(), "Usage: pmm-admin <command>")
	assert.NotContains(t, stderr.String(), "panic")
	assert.NotContains(t, stderr.String(), "SIGSEGV")
}

func TestVersionJson(t *testing.T) {
	cmd := exec.CommandContext(t.Context(), "pmm-admin", "--version", "--json")
	b, err := cmd.CombinedOutput()
	require.NoError(t, err, "%s", b)

	var jsonStruct any
	err = json.Unmarshal(b, &jsonStruct)
	if err != nil {
		t.Errorf("pmm-admin --version --json produces incorrect output format")
	}
}
