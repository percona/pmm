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

	agentv1 "github.com/percona/pmm/api/agent/v1"
)

func TestPTMySQLSummaryActionRun(t *testing.T) {
	t.Parallel()

	id := "6a479303-5081-46d0-baa0-87d6248c987b"
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

	p := NewPTMySQLSummaryAction("14b2422d-32ec-44fb-9019-8b70e3cc8a3a", time.Second, "sleep", &agentv1.StartActionRequest_PTMySQLSummaryParams{})

	ctx, cancel := context.WithTimeout(context.Background(), p.Timeout())
	time.AfterFunc(time.Millisecond, cancel)
	_, err := p.Run(ctx)

	assert.Error(t, err)
}

func TestBuildMyCnfConfig(t *testing.T) {
	type testParams struct {
		Params   *agentv1.StartActionRequest_PTMySQLSummaryParams
		Expected string
		WantErr  error
	}

	testCases := []testParams{
		{
			Params:   &agentv1.StartActionRequest_PTMySQLSummaryParams{Host: "10.20.30.40", Port: 555, Socket: "/tmp/foo.sock", Username: "person", Password: "secret"},
			Expected: "[client]\n\n\nuser=person\npassword=secret\nsocket=/tmp/foo.sock\n",
		},
		{
			Params:   &agentv1.StartActionRequest_PTMySQLSummaryParams{Host: "10.20.30.40", Port: 555, Socket: "", Username: "person", Password: "secret"},
			Expected: "[client]\nhost=10.20.30.40\nport=555\nuser=person\npassword=secret\n\n",
		},
		{
			Params:   &agentv1.StartActionRequest_PTMySQLSummaryParams{Host: "10.20.30.40", Port: 555, Socket: "/tmp/10.sock", Username: "person", Password: ""},
			Expected: "[client]\n\n\nuser=person\n\nsocket=/tmp/10.sock\n",
		},
		{
			Params:   &agentv1.StartActionRequest_PTMySQLSummaryParams{Host: "10.20.30.40", Port: 555, Socket: "", Username: "", Password: "secret"},
			Expected: "[client]\nhost=10.20.30.40\nport=555\n\npassword=secret\n\n",
		},
		{
			Params:   &agentv1.StartActionRequest_PTMySQLSummaryParams{Host: "10.20.30.40", Port: 65536, Socket: "", Username: "", Password: "secret"},
			Expected: "[client]\nhost=10.20.30.40\n\n\npassword=secret\n\n",
		},
		{
			Params:   &agentv1.StartActionRequest_PTMySQLSummaryParams{Host: "", Port: 555, Socket: "", Username: "", Password: "secret"},
			Expected: "[client]\n\nport=555\n\npassword=secret\n\n",
		},
		{
			Params:   &agentv1.StartActionRequest_PTMySQLSummaryParams{Host: "", Port: 0, Socket: "", Username: "", Password: ""},
			Expected: "[client]\n\n\n\n\n\n",
		},
		{
			Params:   &agentv1.StartActionRequest_PTMySQLSummaryParams{Host: "", Port: 0, Socket: "", Username: "王华", Password: `"`},
			Expected: "[client]\n\n\nuser=王华\npassword=&#34;\n\n",
		},
		{
			Params:  &agentv1.StartActionRequest_PTMySQLSummaryParams{Socket: "/tmp/mysqld.sock", Username: "test-user\r", Password: "test-password"},
			WantErr: fmt.Errorf("invalid parameters: %w", ErrInvalidCharacter),
		},
	}

	for i, tc := range testCases {
		a := ptMySQLSummaryAction{
			params: tc.Params,
		}
		t.Run(fmt.Sprintf(t.Name()+" %d", i), func(t *testing.T) {
			s, err := a.buildMyCnfConfig()
			if tc.WantErr != nil {
				require.Error(t, err)
				assert.Equal(t, tc.WantErr.Error(), err.Error())
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.Expected, s)
		})
	}
}
