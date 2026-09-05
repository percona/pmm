// Copyright (C) 2023 Percona LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package managedapi_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/genproto/googleapis/api/annotations"
	"google.golang.org/protobuf/proto"

	serverv1 "github.com/percona/pmm/api/server/v1"
	"github.com/percona/pmm/utils/managedapi"
)

// LeaderHealthCheckPath is a hand-written copy of a value the proto owns: the route comes from
// the google.api.http annotation on ServerService.LeaderHealthCheck, and grpc-gateway binds it
// to an unexported pattern, so there is nothing to reference at compile time. This reads the
// annotation back off the generated descriptor, which is what makes the copy safe to keep.
//
// The test lives in an external package so that these imports never become imports of
// managedapi itself, which supervisord, grafana and qan-api2 all depend on.
func TestLeaderHealthCheckPathMatchesTheProto(t *testing.T) {
	t.Parallel()

	service := serverv1.File_server_v1_server_proto.Services().ByName("ServerService")
	require.NotNil(t, service, "ServerService is missing from the descriptor")

	method := service.Methods().ByName("LeaderHealthCheck")
	require.NotNil(t, method, "LeaderHealthCheck is missing from ServerService")

	// A descriptor that stopped carrying the annotation is itself the drift worth catching,
	// so this fails rather than skipping.
	rule, ok := proto.GetExtension(method.Options(), annotations.E_Http).(*annotations.HttpRule)
	require.True(t, ok, "LeaderHealthCheck has no google.api.http annotation")
	require.NotNil(t, rule)

	assert.Equal(t, managedapi.LeaderHealthCheckPath, rule.GetGet())
}
