// Copyright 2019 Percona LLC
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
	"encoding/json"
	"os/exec"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPackages(t *testing.T) {
	cmd := exec.Command("pmm-admin", "-h")
	b, err := cmd.CombinedOutput()
	require.NoError(t, err, "%s", b)

	out := string(b)
	assert.False(t, strings.Contains(out, "httptest.serve"), `pmm-admin should not import package "net/http/httptest"`)
	assert.False(t, strings.Contains(out, "test.run"), `pmm-admin should not import package "testing"`)
}

func TestVersionPlain(t *testing.T) {
	cmd := exec.Command("pmm-admin", "--version")
	b, err := cmd.CombinedOutput()
	require.NoError(t, err, "%s", b)

	out := string(b)
	assert.True(t, strings.Contains(out, `Version:`), `--version output is incorrect"`)
}

func TestVersionJson(t *testing.T) {
	cmd := exec.Command("pmm-admin", "--version", "--json")
	b, err := cmd.CombinedOutput()
	require.NoError(t, err, "%s", b)

	var jsonStruct interface{}
	if err := json.Unmarshal(b, &jsonStruct); err != nil {
		t.Errorf("pmm-admin --version --json produces incorrect output format")
	}
}
