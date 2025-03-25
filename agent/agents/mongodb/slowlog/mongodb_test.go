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

package slowlog

import (
	"context"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/percona/pmm/agent/utils/templates"
	"github.com/percona/pmm/agent/utils/tests"
	inventoryv1 "github.com/percona/pmm/api/inventory/v1"
)

func TestMongoRun(t *testing.T) {
	sslDSNTemplate, files := tests.GetTestMongoDBWithSSLDSN(t, "../../..")
	tempDir := t.TempDir()
	sslDSN, err := templates.RenderDSN(sslDSNTemplate, files, tempDir)
	require.NoError(t, err)
	for _, params := range []*Params{
		{
			DSN:     "mongodb://root:root-password@127.0.0.1:27017/admin",
			AgentID: "test",
		},
		{
			DSN:     sslDSN,
			AgentID: "test",
		},
	} {
		m, err := New(params, logrus.WithField("test", t.Name()))
		require.NoError(t, err)

		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		go m.Run(ctx)

		// collect only status changes, skip QAN data
		var actual []inventoryv1.AgentStatus
		for c := range m.Changes() {
			if c.Status != inventoryv1.AgentStatus_AGENT_STATUS_UNSPECIFIED {
				actual = append(actual, c.Status)
			}
		}

		expected := []inventoryv1.AgentStatus{
			inventoryv1.AgentStatus_AGENT_STATUS_STARTING,
			inventoryv1.AgentStatus_AGENT_STATUS_RUNNING,
			inventoryv1.AgentStatus_AGENT_STATUS_STOPPING,
			inventoryv1.AgentStatus_AGENT_STATUS_DONE,
		}
		assert.Equal(t, expected, actual)
	}
}
