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
	"testing"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/pkg/errors"
	"github.com/stretchr/objx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/mongo"

	"github.com/percona/pmm/agent/utils/tests"
	agentv1 "github.com/percona/pmm/api/agent/v1"
)

func TestMongoDBActions(t *testing.T) {
	t.Parallel()

	dsn := tests.GetTestMongoDBDSN(t)

	t.Run("getParameter", func(t *testing.T) {
		t.Parallel()
		b := runAction(t, "", 0, dsn, nil, "getParameter", "*", t.TempDir())
		getParameterAssertions(t, b)
	})

	t.Run("buildInfo", func(t *testing.T) {
		t.Parallel()
		b := runAction(t, "", 0, dsn, nil, "buildInfo", 1, t.TempDir())
		buildInfoAssertions(t, b)
	})

	t.Run("getCmdLineOpts", func(t *testing.T) {
		t.Parallel()
		b := runAction(t, "", 0, dsn, nil, "getCmdLineOpts", 1, t.TempDir())
		getCmdLineOptsAssertionsWithAuth(t, b)
	})

	t.Run("replSetGetStatus", func(t *testing.T) {
		t.Parallel()
		replSetGetStatusAssertionsStandalone(t, "", 0, dsn, nil, "replSetGetStatus", 1, t.TempDir())
	})

	t.Run("getDiagnosticData", func(t *testing.T) {
		t.Parallel()
		b := runAction(t, "", 0, dsn, nil, "getDiagnosticData", 1, t.TempDir())
		getDiagnosticDataAssertions(t, b)
	})
}

func TestMongoDBActionsWithSSL(t *testing.T) {
	t.Parallel()

	dsn, files := tests.GetTestMongoDBWithSSLDSN(t, "../../")

	t.Run("getParameter", func(t *testing.T) {
		t.Parallel()
		b := runAction(t, "", 0, dsn, files, "getParameter", "*", t.TempDir())
		getParameterAssertions(t, b)
	})

	t.Run("buildInfo", func(t *testing.T) {
		t.Parallel()
		b := runAction(t, "", 0, dsn, files, "buildInfo", 1, t.TempDir())
		buildInfoAssertions(t, b)
	})

	t.Run("getCmdLineOpts", func(t *testing.T) {
		t.Parallel()
		b := runAction(t, "", 0, dsn, files, "getCmdLineOpts", 1, t.TempDir())
		getCmdLineOptsAssertionsWithSSL(t, b)
	})

	t.Run("replSetGetStatus", func(t *testing.T) {
		t.Parallel()
		replSetGetStatusAssertionsStandalone(t, "", 0, dsn, files, "replSetGetStatus", 1, t.TempDir())
	})

	t.Run("getDiagnosticData", func(t *testing.T) {
		t.Parallel()
		b := runAction(t, "", 0, dsn, files, "getDiagnosticData", 1, t.TempDir())
		getDiagnosticDataAssertions(t, b)
	})
}

func TestMongoDBActionsReplNoAuth(t *testing.T) {
	t.Parallel()

	dsn := tests.GetTestMongoDBReplicatedDSN(t)

	t.Run("getParameter", func(t *testing.T) {
		t.Parallel()
		b := runAction(t, "", 0, dsn, nil, "getParameter", "*", t.TempDir())
		getParameterAssertions(t, b)
	})

	t.Run("buildInfo", func(t *testing.T) {
		t.Parallel()
		b := runAction(t, "", 0, dsn, nil, "buildInfo", 1, t.TempDir())
		buildInfoAssertions(t, b)
	})

	t.Run("getCmdLineOpts", func(t *testing.T) {
		t.Parallel()
		b := runAction(t, "", 0, dsn, nil, "getCmdLineOpts", 1, t.TempDir())
		getCmdLineOptsAssertionsWithoutAuth(t, b)
	})

	t.Run("replSetGetStatus", func(t *testing.T) {
		t.Parallel()
		b := runAction(t, "", 0, dsn, nil, "replSetGetStatus", 1, t.TempDir())
		replSetGetStatusAssertionsReplicated(t, b)
	})

	t.Run("getDiagnosticData", func(t *testing.T) {
		t.Parallel()
		b := runAction(t, "", 0, dsn, nil, "getDiagnosticData", 1, t.TempDir())
		getDiagnosticDataAssertions(t, b)
	})
}

func TestMongoDBActionsReplWithSSL(t *testing.T) {
	t.Parallel()

	dsn, files := tests.GetTestMongoDBReplicatedWithSSLDSN(t, "../../")

	t.Run("getParameter", func(t *testing.T) {
		t.Parallel()
		b := runAction(t, "", 0, dsn, files, "getParameter", "*", t.TempDir())
		getParameterAssertions(t, b)
	})

	t.Run("buildInfo", func(t *testing.T) {
		t.Parallel()
		b := runAction(t, "", 0, dsn, files, "buildInfo", 1, t.TempDir())
		buildInfoAssertions(t, b)
	})

	t.Run("getCmdLineOpts", func(t *testing.T) {
		t.Parallel()
		b := runAction(t, "", 0, dsn, files, "getCmdLineOpts", 1, t.TempDir())
		getCmdLineOptsAssertionsWithSSL(t, b)
	})

	t.Run("replSetGetStatus", func(t *testing.T) {
		t.Parallel()
		b := runAction(t, "", 0, dsn, files, "replSetGetStatus", 1, t.TempDir())
		replSetGetStatusAssertionsReplicated(t, b)
	})

	t.Run("getDiagnosticData", func(t *testing.T) {
		t.Parallel()
		b := runAction(t, "", 0, dsn, files, "getDiagnosticData", 1, t.TempDir())
		getDiagnosticDataAssertions(t, b)
	})
}

func runAction(t *testing.T, id string, timeout time.Duration, dsn string, files *agentv1.TextFiles, command string, arg interface{}, tempDir string) []byte { //nolint:unparam
	t.Helper()
	a, err := NewMongoDBQueryAdmincommandAction(id, timeout, dsn, files, command, arg, tempDir)
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	b, err := a.Run(ctx)
	require.NoError(t, err)
	return b
}

func convertToObjxMap(t *testing.T, b []byte) objx.Map {
	t.Helper()
	data, err := agentv1.UnmarshalActionQueryResult(b)
	require.NoError(t, err)
	t.Log(spew.Sdump(data))
	assert.Len(t, data, 1)
	return data[0]
}

func getParameterAssertions(t *testing.T, b []byte) { //nolint:thelper
	assert.LessOrEqual(t, 5000, len(b))
	objxM := convertToObjxMap(t, b)
	assert.Equal(t, 1.0, objxM.Get("ok").Data())
	assert.Contains(t, objxM.Get("authenticationMechanisms").Data(), "SCRAM-SHA-1")
}

func buildInfoAssertions(t *testing.T, b []byte) { //nolint:thelper
	assert.LessOrEqual(t, 1000, len(b))
	objxM := convertToObjxMap(t, b)
	assert.Equal(t, 1.0, objxM.Get("ok").Data())
	assert.Equal(t, "mozjs", objxM.Get("javascriptEngine").Data())
	assert.Equal(t, "x86_64", objxM.Get("buildEnvironment.distarch").Data())
}

func getDiagnosticDataAssertions(t *testing.T, b []byte) { //nolint:thelper
	assert.LessOrEqual(t, 25000, len(b))
	objxM := convertToObjxMap(t, b)
	assert.Equal(t, 1.0, objxM.Get("ok").Data())
	assert.Equal(t, 1.0, objxM.Get("data.serverStatus.ok").Data())
	assert.Equal(t, "mongod", objxM.Get("data.serverStatus.process").Data())
}

func replSetGetStatusAssertionsReplicated(t *testing.T, b []byte) { //nolint:thelper
	assert.LessOrEqual(t, 1000, len(b))
	objxM := convertToObjxMap(t, b)
	assert.Equal(t, 1.0, objxM.Get("ok").Data())
	assert.Len(t, objxM.Get("members").Data(), 2)
}

func replSetGetStatusAssertionsStandalone(t *testing.T, id string, timeout time.Duration, dsn string, files *agentv1.TextFiles, command string, arg interface{}, tempDir string) { //nolint:thelper
	a, err := NewMongoDBQueryAdmincommandAction(id, timeout, dsn, files, command, arg, tempDir)
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	b, err := a.Run(ctx)
	require.Nil(t, b)
	require.IsType(t, mongo.CommandError{}, errors.Unwrap(err))
	require.Equal(t, "(NoReplicationEnabled) not running with --replSet", err.Error())
}

func getCmdLineOptsAssertionsWithAuth(t *testing.T, b []byte) { //nolint:thelper
	objxM := convertToObjxMap(t, b)
	assert.Equal(t, "1", objxM.Get("ok").String())
	parsed := objxM.Get("parsed").ObjxMap()
	operationProfiling := parsed.Get("operationProfiling").ObjxMap()
	assert.Len(t, operationProfiling, 1)
	assert.Equal(t, "all", operationProfiling.Get("mode").String())

	security := parsed.Get("security").ObjxMap()
	assert.Len(t, security, 1)
	assert.Equal(t, "enabled", security.Get("authorization").String())

	argv := objxM.Get("argv").InterSlice()
	for _, v := range []interface{}{"mongod", "--profile=2", "--auth"} {
		assert.Contains(t, argv, v)
	}
}

func getCmdLineOptsAssertionsWithoutAuth(t *testing.T, b []byte) { //nolint:thelper
	objxM := convertToObjxMap(t, b)
	assert.Equal(t, "1", objxM.Get("ok").String())
	parsed := objxM.Get("parsed").ObjxMap()
	operationProfiling := parsed.Get("operationProfiling").ObjxMap()
	assert.Len(t, operationProfiling, 1)
	assert.Equal(t, "all", operationProfiling.Get("mode").String())

	security := parsed.Get("security").ObjxMap()
	assert.Len(t, security, 1)
	assert.Equal(t, "disabled", security.Get("authorization").String())

	argv := objxM.Get("argv").InterSlice()
	for _, v := range []interface{}{"mongod", "--profile=2", "--noauth"} {
		assert.Contains(t, argv, v)
	}
}

func getCmdLineOptsAssertionsWithSSL(t *testing.T, b []byte) { //nolint:thelper
	objxM := convertToObjxMap(t, b)
	assert.Equal(t, "1", objxM.Get("ok").String())
	parsed := objxM.Get("parsed").ObjxMap()
	operationProfiling := parsed.Get("operationProfiling").ObjxMap()
	assert.Len(t, operationProfiling, 1)
	assert.Equal(t, "all", operationProfiling.Get("mode").String())

	security := parsed.Get("security").ObjxMap()
	assert.Len(t, security, 0)

	argv := objxM.Get("argv").InterSlice()
	expected := []interface{}{"mongod", "--tlsMode=requireTLS", "--tlsCertificateKeyFile=/etc/tls/certificates/server.pem"}
	assert.Subset(t, argv, expected)
}
