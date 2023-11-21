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

package inventory

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/percona/pmm/api/inventorypb/json/client/nodes"
)

func TestAddNodeRemoteRDS(t *testing.T) {
	res := &addNodeRemoteRDSResult{
		Node: &nodes.AddNodeOKBodyRemoteRDS{
			NodeID:       "/node_id/1",
			NodeName:     "rds1",
			Address:      "rds-mysql57",
			NodeModel:    "db.t3.micro",
			Region:       "us-east-1",
			Az:           "us-east-1b",
			CustomLabels: map[string]string{"foo": "bar"},
		},
	}
	expected := strings.TrimSpace(`
Remote RDS Node added.
Node ID  : /node_id/1
Node name: rds1

Address       : rds-mysql57
Model         : db.t3.micro
Custom labels : map[foo:bar]

Region    : us-east-1
Az        : us-east-1b
	`)
	assert.Equal(t, expected, strings.TrimSpace(res.String()))
}
