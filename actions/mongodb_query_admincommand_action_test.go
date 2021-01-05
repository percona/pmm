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
	"io/ioutil"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/percona/pmm/api/agentpb"
	"github.com/stretchr/objx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/percona/pmm-agent/utils/templates"
	"github.com/percona/pmm-agent/utils/tests"
)

func TestMongoDBBuildinfo(t *testing.T) {
	t.Parallel()

	dsn := tests.GetTestMongoDBDSN(t)
	client := tests.OpenTestMongoDB(t, dsn)
	defer client.Disconnect(context.Background()) //nolint:errcheck

	t.Run("getParameter", func(t *testing.T) {
		a := NewMongoDBQueryAdmincommandAction(MongoDBQueryAdmincommandActionParams{
			ID:      "",
			DSN:     dsn,
			Files:   nil,
			Command: "getParameter",
			Arg:     "*",
			TempDir: os.TempDir(),
		})
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()

		b, err := a.Run(ctx)
		require.NoError(t, err)
		assert.LessOrEqual(t, 5000, len(b))
		assert.LessOrEqual(t, len(b), 12000)

		data, err := agentpb.UnmarshalActionQueryResult(b)
		require.NoError(t, err)
		t.Log(spew.Sdump(data))
		assert.Len(t, data, 1)
		m := objx.Map(data[0])
		assert.Equal(t, 1.0, m.Get("ok").Data())
		assert.Contains(t, m.Get("authenticationMechanisms").Data(), "SCRAM-SHA-1")
	})

	t.Run("buildInfo", func(t *testing.T) {
		a := NewMongoDBQueryAdmincommandAction(MongoDBQueryAdmincommandActionParams{
			ID:      "",
			DSN:     tests.GetTestMongoDBDSN(t),
			Files:   nil,
			Command: "buildInfo",
			Arg:     1,
			TempDir: os.TempDir(),
		})
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()

		b, err := a.Run(ctx)
		require.NoError(t, err)
		assert.LessOrEqual(t, 1000, len(b))
		assert.LessOrEqual(t, len(b), 2000)

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
		a := NewMongoDBQueryAdmincommandAction(MongoDBQueryAdmincommandActionParams{
			ID:      "",
			DSN:     tests.GetTestMongoDBDSN(t),
			Files:   nil,
			Command: "getCmdLineOpts",
			Arg:     1,
			TempDir: os.TempDir(),
		})
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

func TestMongoDBBuildinfoWithSSL(t *testing.T) {
	t.Parallel()

	dsnTemplate, files := tests.GetTestMongoDBWithSSLDSN(t, "../")
	tempDir, err := ioutil.TempDir("", "pmm-agent-")
	require.NoError(t, err)
	dsn, err := templates.RenderDSN(dsnTemplate, files, tempDir)
	require.NoError(t, err)
	client := tests.OpenTestMongoDB(t, dsn)
	defer client.Disconnect(context.Background()) //nolint:errcheck

	t.Run("getParameter", func(t *testing.T) {
		tempDir, err := ioutil.TempDir("", "pmm-agent-")
		require.NoError(t, err)
		a := NewMongoDBQueryAdmincommandAction(MongoDBQueryAdmincommandActionParams{
			ID:      "",
			DSN:     dsnTemplate,
			Command: "getParameter",
			Arg:     "*",
			TempDir: tempDir,
			Files:   files,
		})
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()

		b, err := a.Run(ctx)
		require.NoError(t, err)
		assert.LessOrEqual(t, 5000, len(b))
		assert.LessOrEqual(t, len(b), 12000)

		data, err := agentpb.UnmarshalActionQueryResult(b)
		require.NoError(t, err)
		t.Log(spew.Sdump(data))
		assert.Len(t, data, 1)
		m := objx.Map(data[0])
		assert.Equal(t, 1.0, m.Get("ok").Data())
		assert.Contains(t, m.Get("authenticationMechanisms").Data(), "SCRAM-SHA-1")
	})

	t.Run("buildInfo", func(t *testing.T) {
		tempDir, err := ioutil.TempDir("", "pmm-agent-")
		require.NoError(t, err)
		a := NewMongoDBQueryAdmincommandAction(MongoDBQueryAdmincommandActionParams{
			ID:      "",
			DSN:     dsnTemplate,
			Command: "buildInfo",
			Arg:     1,
			TempDir: tempDir,
			Files:   files,
		})
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()

		b, err := a.Run(ctx)
		require.NoError(t, err)
		assert.LessOrEqual(t, 1000, len(b))
		assert.LessOrEqual(t, len(b), 2000)

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
		tempDir, err := ioutil.TempDir("", "pmm-agent-")
		require.NoError(t, err)
		a := NewMongoDBQueryAdmincommandAction(MongoDBQueryAdmincommandActionParams{
			ID:      "",
			DSN:     dsnTemplate,
			Command: "getCmdLineOpts",
			Arg:     1,
			TempDir: tempDir,
			Files:   files,
		})
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()

		b, err := a.Run(ctx)
		require.NoError(t, err)

		data, err := agentpb.UnmarshalActionQueryResult(b)
		require.NoError(t, err)
		assert.Len(t, data, 1)
		t.Log(spew.Sdump(data))

		m := objx.Map(data[0])

		parsed := m.Get("parsed").ObjxMap()

		operationProfiling := parsed.Get("operationProfiling").ObjxMap()
		assert.Len(t, operationProfiling, 1)
		assert.Equal(t, "all", operationProfiling.Get("mode").String())

		security := parsed.Get("security").ObjxMap()
		assert.Len(t, security, 0)

		argv := m.Get("argv").InterSlice()
		expected := []interface{}{"mongod", "--sslMode=requireSSL", "--sslPEMKeyFile=/etc/ssl/certificates/server.pem"}

		var tlsMode bool
		for _, arg := range argv {
			if strings.Contains(arg.(string), "tlsMode") {
				tlsMode = true
				break
			}
		}
		if tlsMode {
			expected = []interface{}{"mongod", "--tlsMode", "requireTLS", "--tlsCertificateKeyFile", "/etc/ssl/certificates/server.pem"}
		}
		assert.Subset(t, argv, expected)

		assert.Equal(t, "1", m.Get("ok").String())
	})
}
