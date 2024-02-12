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
	"fmt"
	"math/rand"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"

	"github.com/percona/pmm/api/inventorypb/json/client"
	"github.com/percona/pmm/api/inventorypb/json/client/agents"
	"github.com/percona/pmm/api/inventorypb/json/client/nodes"
	"github.com/percona/pmm/api/inventorypb/json/client/services"
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

	n := rand.Int() //nolint:gosec
	return fmt.Sprintf("pmm-api-tests/%s/%s/%s/%d", Hostname, t.Name(), name, n)
}

// AssertAPIErrorf check that actual API error equals expected.
func AssertAPIErrorf(t TestingT, actual error, httpStatus int, grpcCode codes.Code, format string, a ...interface{}) {
	t.Helper()

	require.Error(t, actual)

	require.Implementsf(t, (*ErrorResponse)(nil), actual, "Wrong response type. Expected %T, got %T.\nError message: %v", (*ErrorResponse)(nil), actual, actual)

	assert.Equal(t, httpStatus, actual.(ErrorResponse).Code()) //nolint:forcetypeassert,errorlint

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

// RemoveNodes removes specified nodes.
func RemoveNodes(t TestingT, nodeIDs ...string) {
	t.Helper()

	for _, nodeID := range nodeIDs {
		params := &nodes.RemoveNodeParams{
			Body: nodes.RemoveNodeBody{
				NodeID: nodeID,
			},
			Context: context.Background(),
		}
		res, err := client.Default.Nodes.RemoveNode(params)
		assert.NoError(t, err)
		assert.NotNil(t, res)
	}
}

// RemoveServices removes specified services.
func RemoveServices(t TestingT, serviceIDs ...string) {
	t.Helper()

	for _, serviceID := range serviceIDs {
		params := &services.RemoveServiceParams{
			Body: services.RemoveServiceBody{
				ServiceID: serviceID,
				Force:     true,
			},
			Context: context.Background(),
		}
		res, err := client.Default.Services.RemoveService(params)
		assert.NoError(t, err)
		assert.NotNil(t, res)
	}
}

// RemoveAgents removes specified agents.
func RemoveAgents(t TestingT, agentIDs ...string) {
	t.Helper()

	for _, agentID := range agentIDs {
		params := &agents.RemoveAgentParams{
			Body: agents.RemoveAgentBody{
				AgentID: agentID,
			},
			Context: context.Background(),
		}
		res, err := client.Default.Agents.RemoveAgent(params)
		assert.NoError(t, err)
		assert.NotNil(t, res)
	}
}

// AddGenericNode adds a generic node.
func AddGenericNode(t TestingT, nodeName string) *nodes.AddGenericNodeOKBodyGeneric {
	t.Helper()

	params := &nodes.AddGenericNodeParams{
		Body: nodes.AddGenericNodeBody{
			NodeName: nodeName,
			Address:  "10.10.10.10",
		},
		Context: Context,
	}
	res, err := client.Default.Nodes.AddGenericNode(params)
	assert.NoError(t, err)
	require.NotNil(t, res)
	require.NotNil(t, res.Payload)
	require.NotNil(t, res.Payload.Generic)
	return res.Payload.Generic
}

// AddRemoteNode adds a remote node.
func AddRemoteNode(t TestingT, nodeName string) *nodes.AddRemoteNodeOKBody {
	t.Helper()

	params := &nodes.AddRemoteNodeParams{
		Body: nodes.AddRemoteNodeBody{
			NodeName: nodeName,
			Address:  "10.10.10.10",
		},
		Context: Context,
	}
	res, err := client.Default.Nodes.AddRemoteNode(params)
	assert.NoError(t, err)
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

	res, err := client.Default.Nodes.AddNode(params)
	assert.NoError(t, err)
	require.NotNil(t, res)

	return res.Payload
}

// AddPMMAgent adds a PMM agent.
func AddPMMAgent(t TestingT, nodeID string) *agents.AddPMMAgentOKBody {
	t.Helper()

	res, err := client.Default.Agents.AddPMMAgent(&agents.AddPMMAgentParams{
		Body: agents.AddPMMAgentBody{
			RunsOnNodeID: nodeID,
		},
		Context: Context,
	})
	assert.NoError(t, err)
	require.NotNil(t, res)
	return res.Payload
}

// check interfaces.
var (
	_ assert.TestingT  = (*expectedFailureTestingT)(nil)
	_ require.TestingT = (*expectedFailureTestingT)(nil)
	_ TestingT         = (*expectedFailureTestingT)(nil)
)
