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

package actions

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	agentpb "github.com/percona/pmm/api/agentpb/v1"
)

func TestPTMySQLSummaryActionRun(t *testing.T) {
	t.Parallel()

	id := "/action_id/6a479303-5081-46d0-baa0-87d6248c987b"
	cmd := "echo"
	p := NewPTMySQLSummaryAction(id, 5*time.Second, cmd, nil)

	ctx, cancel := context.WithTimeout(context.Background(), p.Timeout())
	defer cancel()
	got, err := p.Run(ctx)

	require.NoError(t, err)
	assert.NotEmpty(t, got)
	assert.Equal(t, id, p.ID())
	assert.Equal(t, cmd, p.Type())
}

func TestPTMySQLSummaryActionRunAndCancel(t *testing.T) {
	t.Parallel()

	p := NewPTMySQLSummaryAction("/action_id/14b2422d-32ec-44fb-9019-8b70e3cc8a3a", time.Second, "sleep", &agentpb.StartActionRequest_PTMySQLSummaryParams{})

	ctx, cancel := context.WithTimeout(context.Background(), p.Timeout())
	time.AfterFunc(time.Millisecond, cancel)
	_, err := p.Run(ctx)

	assert.Error(t, err)
}

func TestListFromMySqlParams(t *testing.T) {
	type testParams struct {
		Params   *agentpb.StartActionRequest_PTMySQLSummaryParams
		Expected []string
	}

	testCases := []testParams{
		{
			Params:   &agentpb.StartActionRequest_PTMySQLSummaryParams{Host: "10.20.30.40", Port: 555, Socket: "10", Username: "person", Password: "secret"},
			Expected: []string{"--socket", "10", "--user", "person", "--password", "secret"},
		},
		{
			Params:   &agentpb.StartActionRequest_PTMySQLSummaryParams{Host: "10.20.30.40", Port: 555, Socket: "", Username: "person", Password: "secret"},
			Expected: []string{"--host", "10.20.30.40", "--port", "555", "--user", "person", "--password", "secret"},
		},
		{
			Params:   &agentpb.StartActionRequest_PTMySQLSummaryParams{Host: "10.20.30.40", Port: 555, Socket: "10", Username: "person", Password: ""},
			Expected: []string{"--socket", "10", "--user", "person"},
		},
		{
			Params:   &agentpb.StartActionRequest_PTMySQLSummaryParams{Host: "10.20.30.40", Port: 555, Socket: "", Username: "", Password: "secret"},
			Expected: []string{"--host", "10.20.30.40", "--port", "555", "--password", "secret"},
		},
		{
			Params:   &agentpb.StartActionRequest_PTMySQLSummaryParams{Host: "10.20.30.40", Port: 65536, Socket: "", Username: "", Password: "secret"},
			Expected: []string{"--host", "10.20.30.40", "--password", "secret"},
		},
		{
			Params:   &agentpb.StartActionRequest_PTMySQLSummaryParams{Host: "", Port: 555, Socket: "", Username: "", Password: "secret"},
			Expected: []string{"--port", "555", "--password", "secret"},
		},
		{
			Params:   &agentpb.StartActionRequest_PTMySQLSummaryParams{Host: "", Port: 0, Socket: "", Username: "", Password: ""},
			Expected: []string{},
		},
		{
			Params:   &agentpb.StartActionRequest_PTMySQLSummaryParams{Host: "", Port: 0, Socket: "", Username: "王华", Password: `"`},
			Expected: []string{"--user", "王华", "--password", `"`},
		},
	}

	for i, tc := range testCases {
		a := ptMySQLSummaryAction{
			params: tc.Params,
		}
		t.Run(fmt.Sprintf("TestListFromMySqlParams %d", i), func(t *testing.T) {
			assert.ElementsMatch(t, tc.Expected, a.ListFromMySQLParams())
		})
	}
}
