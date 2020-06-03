// pmm-agent
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

package actions

import (
	"context"
	"testing"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/percona/pmm/api/agentpb"
	"github.com/stretchr/objx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/percona/pmm-agent/utils/tests"
)

func TestMongoDBBuildinfo(t *testing.T) {
	t.Parallel()

	client := tests.OpenTestMongoDB(t)
	defer client.Disconnect(context.Background()) //nolint:errcheck

	t.Run("getParameter", func(t *testing.T) {
		a := NewMongoDBQueryAdmincommandAction("", tests.GetTestMongoDBDSN(t), "getParameter", "*")
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()

		b, err := a.Run(ctx)
		require.NoError(t, err)
		assert.LessOrEqual(t, 5433, len(b))
		assert.LessOrEqual(t, len(b), 10608)

		data, err := agentpb.UnmarshalActionQueryResult(b)
		require.NoError(t, err)
		t.Log(spew.Sdump(data))
		assert.Len(t, data, 1)
		m := objx.Map(data[0])
		assert.Equal(t, 1.0, m.Get("ok").Data())
		assert.Contains(t, m.Get("authenticationMechanisms").Data(), "SCRAM-SHA-1")
	})

	t.Run("buildInfo", func(t *testing.T) {
		a := NewMongoDBQueryAdmincommandAction("", tests.GetTestMongoDBDSN(t), "buildInfo", 1)
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()

		b, err := a.Run(ctx)
		require.NoError(t, err)
		assert.LessOrEqual(t, 1262, len(b))
		assert.LessOrEqual(t, len(b), 1446)

		data, err := agentpb.UnmarshalActionQueryResult(b)
		require.NoError(t, err)
		t.Log(spew.Sdump(data))
		assert.Len(t, data, 1)
		m := objx.Map(data[0])
		assert.Equal(t, 1.0, m.Get("ok").Data())
		assert.Equal(t, "mozjs", m.Get("javascriptEngine").Data())
		assert.Equal(t, "x86_64", m.Get("buildEnvironment.distarch").Data())
	})

	t.Run("getCmdLineOpts", func(t *testing.T) {
		a := NewMongoDBQueryAdmincommandAction("", tests.GetTestMongoDBDSN(t), "getCmdLineOpts", 1)
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()

		b, err := a.Run(ctx)
		require.NoError(t, err)

		data, err := agentpb.UnmarshalActionQueryResult(b)
		require.NoError(t, err)
		assert.Len(t, data, 1)
		t.Log(spew.Sdump(data))

		m := objx.Map(data[0])

		argv := m.Get("argv").InterSlice()
		for _, v := range []interface{}{"mongod", "--profile", "2", "--auth"} {
			assert.Contains(t, argv, v)
		}

		parsed := m.Get("parsed").ObjxMap()

		operationProfiling := parsed.Get("operationProfiling").ObjxMap()
		assert.Len(t, operationProfiling, 1)
		assert.Equal(t, "all", operationProfiling.Get("mode").String())

		security := parsed.Get("security").ObjxMap()
		assert.Len(t, security, 1)
		assert.Equal(t, "enabled", security.Get("authorization").String())

		assert.Equal(t, "1", m.Get("ok").String())
	})
}
