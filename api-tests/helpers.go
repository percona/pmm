// Copyright (C) 2023 Percona LLC
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program. If not, see <https://www.gnu.org/licenses/>.

package apitests

import (
	"context"
	"crypto/rand"
	"fmt"
	"math"
	"math/big"
	"reflect"
	"strings"
	"testing"

	"github.com/AlekSi/pointer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"

	"github.com/percona/pmm/api/inventory/v1/json/client"
	agents "github.com/percona/pmm/api/inventory/v1/json/client/agents_service"
	nodes "github.com/percona/pmm/api/inventory/v1/json/client/nodes_service"
	services "github.com/percona/pmm/api/inventory/v1/json/client/services_service"
	managementClient "github.com/percona/pmm/api/management/v1/json/client"
	mservice "github.com/percona/pmm/api/management/v1/json/client/management_service"
)

// ErrorResponse represents the response structure for error scenarios.
type ErrorResponse interface {
	Code() int
}

// TestingT contains minimal subset of *testing.T that we use that is also should be implemented by *expectedFailureTestingT.
type TestingT interface {
	Helper()
	Name() string
	Errorf(format string, args ...interface{})
	FailNow()
}

// TestString returns semi-random string that can be used as a test data.
func TestString(t TestingT, name string) string {
	t.Helper()

	// Without proper seed parallel tests can generate same "random" number.
	rnd, err := rand.Int(rand.Reader, big.NewInt(math.MaxInt32))
	require.NoError(t, err)

	return strings.ReplaceAll(fmt.Sprintf("api-test-%s-%s-%d", t.Name(), name, rnd), "/", "-")
}

// AssertAPIErrorf check that actual API error equals expected.
func AssertAPIErrorf(t TestingT, actual error, httpStatus int, grpcCode codes.Code, format string, a ...interface{}) {
	t.Helper()

	require.Error(t, actual)

	require.Implementsf(t, (*ErrorResponse)(nil), actual, "Wrong response type. Expected %T, got %T.\nError message: %v", (*ErrorResponse)(nil), actual, actual)

	assert.Equal(t, httpStatus, actual.(ErrorResponse).Code()) //nolint:forcetypeassert

	// Have to use reflect because there are a lot of types with the same structure and different names.
	payload := reflect.ValueOf(actual).Elem().FieldByName("Payload")
	require.True(t, payload.IsValid(), "Wrong response structure. There is no field Payload.")

	codeField := payload.Elem().FieldByName("Code")
	require.True(t, codeField.IsValid(), "Wrong response structure. There is no field Code in Payload.")
	assert.Equal(t, int64(grpcCode), codeField.Int(), "gRPC status codes are not equal")

	errorField := payload.Elem().FieldByName("Message")
	require.True(t, errorField.IsValid(), "Wrong response structure. There is no field Message in Payload.")
	if len(a) != 0 {
		format = fmt.Sprintf(format, a...)
	}
	// We use "assert.Contains" because some error messages include info that changes easily
	// (e.g. the line number in the proto file).
	assert.Contains(t, errorField.String(), format)
}

// ExpectFailure sets up expectations for a test case to fail.
func ExpectFailure(t *testing.T, link string) *expectedFailureTestingT { //nolint:revive
	t.Helper()
	return &expectedFailureTestingT{
		t:    t,
		link: link,
	}
}

// expectedFailureTestingT expects that test will fail.
// If the test fails - we skip it,
// if it doesn't - we call Fail.
type expectedFailureTestingT struct {
	t      *testing.T
	errors []string
	failed bool
	link   string
}

func (tt *expectedFailureTestingT) Helper()      { tt.t.Helper() }
func (tt *expectedFailureTestingT) Name() string { return tt.t.Name() }

func (tt *expectedFailureTestingT) Errorf(format string, args ...interface{}) {
	tt.errors = append(tt.errors, fmt.Sprintf(format, args...))
	tt.failed = true
}

func (tt *expectedFailureTestingT) FailNow() {
	tt.failed = true

	// We have to set unexported testing.T.finished = true to make everything work,
	// but we can't call tt.t.FailNow() as it calls Fail().
	tt.t.SkipNow()
}

func (tt *expectedFailureTestingT) Check() {
	tt.t.Helper()

	if tt.failed {
		for _, v := range tt.errors {
			tt.t.Log(v)
		}
		tt.t.Skipf("Expected failure: %s", tt.link)
		return
	}

	tt.t.Fatalf("%s expected to fail, but didn't: %s", tt.Name(), tt.link)
}

// UnregisterNodes unregister specified nodes.
func UnregisterNodes(t TestingT, nodeIDs ...string) {
	t.Helper()

	for _, nodeID := range nodeIDs {
		params := &mservice.UnregisterNodeParams{
			NodeID:  nodeID,
			Force:   pointer.ToBool(true),
			Context: context.Background(),
		}

		res, err := managementClient.Default.ManagementService.UnregisterNode(params)
		require.NoError(t, err)
		assert.NotNil(t, res)
		assert.NotNil(t, res.Payload)
		assert.Empty(t, res.Payload.Warning)
	}
}

// RemoveNodes removes specified nodes.
func RemoveNodes(t TestingT, nodeIDs ...string) {
	t.Helper()

	for _, nodeID := range nodeIDs {
		params := &nodes.RemoveNodeParams{
			NodeID:  nodeID,
			Context: context.Background(),
		}
		res, err := client.Default.NodesService.RemoveNode(params)
		require.NoError(t, err)
		assert.NotNil(t, res)
	}
}

// RemoveServices removes specified services.
func RemoveServices(t TestingT, serviceIDs ...string) {
	t.Helper()

	for _, serviceID := range serviceIDs {
		params := &services.RemoveServiceParams{
			ServiceID: serviceID,
			Force:     pointer.ToBool(true),
			Context:   context.Background(),
		}
		res, err := client.Default.ServicesService.RemoveService(params)
		require.NoError(t, err)
		assert.NotNil(t, res)
	}
}

// RemoveAgents removes specified agents.
func RemoveAgents(t TestingT, agentIDs ...string) {
	t.Helper()

	for _, agentID := range agentIDs {
		params := &agents.RemoveAgentParams{
			AgentID: agentID,
			Context: context.Background(),
		}
		res, err := client.Default.AgentsService.RemoveAgent(params)
		require.NoError(t, err)
		assert.NotNil(t, res)
	}
}

// RemoveAgentsWithForce removes specified agents along with dependents.
func RemoveAgentsWithForce(t TestingT, agentIDs ...string) {
	t.Helper()

	for _, agentID := range agentIDs {
		params := &agents.RemoveAgentParams{
			AgentID: agentID,
			Force:   pointer.ToBool(true),
			Context: context.Background(),
		}
		res, err := client.Default.AgentsService.RemoveAgent(params)
		require.NoError(t, err)
		assert.NotNil(t, res)
	}
}

// AddGenericNode adds a generic node.
func AddGenericNode(t TestingT, nodeName string) *nodes.AddNodeOKBodyGeneric {
	t.Helper()

	params := &nodes.AddNodeParams{
		Body: nodes.AddNodeBody{
			Generic: &nodes.AddNodeParamsBodyGeneric{
				NodeName: nodeName,
				Address:  "10.10.10.10",
			},
		},
		Context: Context,
	}
	res, err := client.Default.NodesService.AddNode(params)
	require.NoError(t, err)
	require.NotNil(t, res)
	require.NotNil(t, res.Payload)
	require.NotNil(t, res.Payload.Generic)
	return res.Payload.Generic
}

// AddRemoteNode adds a remote node.
func AddRemoteNode(t TestingT, nodeName string) *nodes.AddNodeOKBody {
	t.Helper()

	params := &nodes.AddNodeParams{
		Body: nodes.AddNodeBody{
			Remote: &nodes.AddNodeParamsBodyRemote{
				NodeName: nodeName,
				Address:  "10.10.10.10",
			},
		},
		Context: Context,
	}
	res, err := client.Default.NodesService.AddNode(params)
	require.NoError(t, err)
	require.NotNil(t, res)
	return res.Payload
}

// AddNode adds a node.
func AddNode(t TestingT, nodeBody *nodes.AddNodeBody) *nodes.AddNodeOKBody {
	t.Helper()

	params := &nodes.AddNodeParams{
		Body:    *nodeBody,
		Context: Context,
	}

	res, err := client.Default.NodesService.AddNode(params)
	require.NoError(t, err)
	require.NotNil(t, res)

	return res.Payload
}

// AddPMMAgent adds a PMM agent.
func AddPMMAgent(t TestingT, nodeID string) *agents.AddAgentOKBody {
	t.Helper()

	res, err := client.Default.AgentsService.AddAgent(&agents.AddAgentParams{
		Body: agents.AddAgentBody{
			PMMAgent: &agents.AddAgentParamsBodyPMMAgent{
				RunsOnNodeID: nodeID,
			},
		},
		Context: Context,
	})
	require.NoError(t, err)
	require.NotNil(t, res)
	return res.Payload
}

// check interfaces.
var (
	_ assert.TestingT  = (*expectedFailureTestingT)(nil)
	_ require.TestingT = (*expectedFailureTestingT)(nil)
	_ TestingT         = (*expectedFailureTestingT)(nil)
)
