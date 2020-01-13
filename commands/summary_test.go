// pmm-admin
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

package commands

import (
	"context"
	"io/ioutil"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/percona/pmm-admin/agentlocal"
)

func TestSummary(t *testing.T) {
	agentlocal.SetTransport(context.TODO(), true)

	f, err := ioutil.TempFile("", "pmm-admin-test-summary")
	defer os.Remove(f.Name())
	require.NoError(t, err)
	assert.NoError(t, f.Close())

	filename := f.Name()
	t.Log(filename)
	t.Run("Summary default", func(t *testing.T) {
		cmd := &summaryCommand{
			Filename: filename,
		}
		res, err := cmd.Run()
		require.NoError(t, err)
		assert.NotNil(t, res)
		assert.NoError(t, os.Remove(filename))
	})

	t.Run("Summary skip server", func(t *testing.T) {
		cmd := &summaryCommand{
			Filename:   filename,
			SkipServer: true,
		}
		res, err := cmd.Run()
		require.NoError(t, err)
		assert.NotNil(t, res)
		assert.NoError(t, os.Remove(filename))
	})
}
