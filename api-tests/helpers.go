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
	"sync"
	"testing"
	"time"

	grafanaclient "github.com/grafana/grafana-openapi-client-go/client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"

	"github.com/percona/pmm/api/inventory/v1/json/client"
	agents "github.com/percona/pmm/api/inventory/v1/json/client/agents_service"
	nodes "github.com/percona/pmm/api/inventory/v1/json/client/nodes_service"
	services "github.com/percona/pmm/api/inventory/v1/json/client/services_service"
	managementClient "github.com/percona/pmm/api/management/v1/json/client"
	mservice "github.com/percona/pmm/api/management/v1/json/client/management_service"
	serverClient "github.com/percona/pmm/api/server/v1/json/client"
	server "github.com/percona/pmm/api/server/v1/json/client/server_service"
	userClient "github.com/percona/pmm/api/user/v1/json/client"
	"github.com/percona/pmm/api/user/v1/json/client/user_service"
)

// ErrorResponse represents the response structure for error scenarios.
type ErrorResponse interface {
	Code() int
}

// TestString returns semi-random string that can be used as a test data.
func TestString(t *testing.T, name string) string {
	t.Helper()

	// Without proper seed parallel tests can generate same "random" number.
	rnd, err := rand.Int(rand.Reader, big.NewInt(math.MaxInt32))
	require.NoError(t, err)

	return strings.ReplaceAll(fmt.Sprintf("api-test-%s-%s-%d", t.Name(), name, rnd), "/", "-")
}

// AssertAPIErrorf check that actual API error equals expected.
func AssertAPIErrorf(t *testing.T, actual error, httpStatus int, grpcCode codes.Code, format string, a ...any) {
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

// UnregisterNodes unregister specified nodes.
func UnregisterNodes(t *testing.T, nodeIDs ...string) {
	t.Helper()

	for _, nodeID := range nodeIDs {
		params := &mservice.UnregisterNodeParams{
			NodeID:  nodeID,
			Force:   new(true),
			Context: context.Background(),
		}

		res, err := managementClient.Default.ManagementService.UnregisterNode(params)
		if err == nil {
			assert.NotNil(t, res)
			assert.NotNil(t, res.Payload)
			assert.Empty(t, res.Payload.Warning)
		}
	}
}

// RemoveNodes removes specified nodes.
func RemoveNodes(t *testing.T, nodeIDs ...string) {
	t.Helper()

	for _, nodeID := range nodeIDs {
		params := &nodes.RemoveNodeParams{
			NodeID:  nodeID,
			Force:   new(true),
			Context: context.Background(),
		}
		res, err := client.Default.NodesService.RemoveNode(params)
		if err == nil {
			assert.NotNil(t, res)
		}
	}
}

// RemoveServices removes specified services.
func RemoveServices(t *testing.T, serviceIDs ...string) {
	t.Helper()

	for _, serviceID := range serviceIDs {
		params := &services.RemoveServiceParams{
			ServiceID: serviceID,
			Force:     new(true),
			Context:   context.Background(),
		}
		res, err := client.Default.ServicesService.RemoveService(params)
		if err == nil {
			assert.NotNil(t, res)
		}
	}
}

// RemoveAgents removes specified agents.
func RemoveAgents(t *testing.T, agentIDs ...string) {
	t.Helper()

	for _, agentID := range agentIDs {
		params := &agents.RemoveAgentParams{
			AgentID: agentID,
			Force:   new(true),
			Context: t.Context(),
		}
		res, err := client.Default.AgentsService.RemoveAgent(params)
		if err == nil {
			assert.NotNil(t, res)
		}
	}
}

// AddGenericNode adds a generic node.
// The node will be automatically removed after the test.
func AddGenericNode(t *testing.T, nodeName string) *nodes.AddNodeOKBodyGeneric {
	t.Helper()

	body := &nodes.AddNodeBody{
		Generic: &nodes.AddNodeParamsBodyGeneric{
			NodeName: nodeName,
			Address:  TestString(t, "10.10.10.10"),
		},
	}
	return AddNode(t, body).Generic
}

// AddRemoteNode adds a remote node.
// The node will be automatically removed after the test.
func AddRemoteNode(t *testing.T, nodeName string) *nodes.AddNodeOKBodyRemote {
	t.Helper()

	body := &nodes.AddNodeBody{
		Remote: &nodes.AddNodeParamsBodyRemote{
			NodeName: nodeName,
			Address:  TestString(t, "10.10.10.10"),
		},
	}
	return AddNode(t, body).Remote
}

// AddRemoteRDSNode adds a remote RDS node.
// The node will be automatically removed after the test.
func AddRemoteRDSNode(t *testing.T, nodeName string) *nodes.AddNodeOKBodyRemoteRDS {
	t.Helper()

	body := &nodes.AddNodeBody{
		RemoteRDS: &nodes.AddNodeParamsBodyRemoteRDS{
			NodeName: nodeName,
			Address:  TestString(t, "rds-address"),
			Region:   TestString(t, "rds-region"),
		},
	}
	return AddNode(t, body).RemoteRDS
}

// AddRemoteAzureNode adds a remote Azure Database node.
// The node will be automatically removed after the test.
func AddRemoteAzureNode(t *testing.T, nodeName string) *nodes.AddNodeOKBodyRemoteAzureDatabase {
	t.Helper()

	body := &nodes.AddNodeBody{
		RemoteAzure: &nodes.AddNodeParamsBodyRemoteAzure{
			NodeName: nodeName,
			Address:  TestString(t, "azure-address"),
			Region:   TestString(t, "azure-region"),
		},
	}
	return AddNode(t, body).RemoteAzureDatabase
}

// AddNode adds a node.
// The node will be automatically removed after the test.
func AddNode(t *testing.T, nodeBody *nodes.AddNodeBody) *nodes.AddNodeOKBody {
	t.Helper()

	params := &nodes.AddNodeParams{
		Body:    *nodeBody,
		Context: t.Context(),
	}

	res, err := client.Default.NodesService.AddNode(params)
	require.NoError(t, err)
	require.NotNil(t, res)
	require.NotNil(t, res.Payload)

	var nodeID string
	switch {
	case nodeBody.Generic != nil:
		require.NotNil(t, res.Payload.Generic)
		nodeID = res.Payload.Generic.NodeID
	case nodeBody.Remote != nil:
		require.NotNil(t, res.Payload.Remote)
		nodeID = res.Payload.Remote.NodeID
	case nodeBody.Container != nil:
		require.NotNil(t, res.Payload.Container)
		nodeID = res.Payload.Container.NodeID
	case nodeBody.RemoteAzure != nil:
		require.NotNil(t, res.Payload.RemoteAzureDatabase)
		nodeID = res.Payload.RemoteAzureDatabase.NodeID
	case nodeBody.RemoteRDS != nil:
		require.NotNil(t, res.Payload.RemoteRDS)
		nodeID = res.Payload.RemoteRDS.NodeID
	}

	require.NotEmpty(t, nodeID)
	t.Cleanup(func() {
		RemoveNodes(t, nodeID)
	})
	return res.Payload
}

// AddPMMAgent adds a PMM agent.
func AddPMMAgent(t *testing.T, nodeID string) *agents.AddAgentOKBodyPMMAgent {
	t.Helper()

	body := agents.AddAgentBody{
		PMMAgent: &agents.AddAgentParamsBodyPMMAgent{
			RunsOnNodeID: nodeID,
		},
	}
	return AddAgent(t, body).PMMAgent
}

// AddNodeExporter adds a Node Exporter agent.
// The agent will be automatically removed after the test.
func AddNodeExporter(t *testing.T, pmmAgentID string, customLabels map[string]string) *agents.AddAgentOKBodyNodeExporter {
	t.Helper()

	body := agents.AddAgentBody{
		NodeExporter: &agents.AddAgentParamsBodyNodeExporter{
			PMMAgentID:   pmmAgentID,
			CustomLabels: customLabels,
		},
	}
	return AddAgent(t, body).NodeExporter
}

// AddAgent adds an agent with the specified body.
// The agent will be automatically removed after the test.
func AddAgent(t *testing.T, body agents.AddAgentBody) *agents.AddAgentOKBody {
	t.Helper()

	res, err := client.Default.AgentsService.AddAgent(&agents.AddAgentParams{
		Body:    body,
		Context: t.Context(),
	})
	require.NoError(t, err)
	require.NotNil(t, res)
	require.NotNil(t, res.Payload)

	var agentID string
	switch {
	case body.PMMAgent != nil:
		require.NotNil(t, res.Payload.PMMAgent)
		agentID = res.Payload.PMMAgent.AgentID
	case body.AzureDatabaseExporter != nil:
		require.NotNil(t, res.Payload.AzureDatabaseExporter)
		agentID = res.Payload.AzureDatabaseExporter.AgentID
	case body.MongodbExporter != nil:
		require.NotNil(t, res.Payload.MongodbExporter)
		agentID = res.Payload.MongodbExporter.AgentID
	case body.ExternalExporter != nil:
		require.NotNil(t, res.Payload.ExternalExporter)
		agentID = res.Payload.ExternalExporter.AgentID
	case body.MysqldExporter != nil:
		require.NotNil(t, res.Payload.MysqldExporter)
		agentID = res.Payload.MysqldExporter.AgentID
	case body.NodeExporter != nil:
		require.NotNil(t, res.Payload.NodeExporter)
		agentID = res.Payload.NodeExporter.AgentID
	case body.PostgresExporter != nil:
		require.NotNil(t, res.Payload.PostgresExporter)
		agentID = res.Payload.PostgresExporter.AgentID
	case body.ProxysqlExporter != nil:
		require.NotNil(t, res.Payload.ProxysqlExporter)
		agentID = res.Payload.ProxysqlExporter.AgentID
	case body.ValkeyExporter != nil:
		require.NotNil(t, res.Payload.ValkeyExporter)
		agentID = res.Payload.ValkeyExporter.AgentID
	case body.RtaMongodbAgent != nil:
		require.NotNil(t, res.Payload.RtaMongodbAgent)
		agentID = res.Payload.RtaMongodbAgent.AgentID
	case body.QANMongodbMongologAgent != nil:
		require.NotNil(t, res.Payload.QANMongodbMongologAgent)
		agentID = res.Payload.QANMongodbMongologAgent.AgentID
	case body.QANMongodbProfilerAgent != nil:
		require.NotNil(t, res.Payload.QANMongodbProfilerAgent)
		agentID = res.Payload.QANMongodbProfilerAgent.AgentID
	case body.QANMysqlPerfschemaAgent != nil:
		require.NotNil(t, res.Payload.QANMysqlPerfschemaAgent)
		agentID = res.Payload.QANMysqlPerfschemaAgent.AgentID
	case body.QANMysqlSlowlogAgent != nil:
		require.NotNil(t, res.Payload.QANMysqlSlowlogAgent)
		agentID = res.Payload.QANMysqlSlowlogAgent.AgentID
	case body.QANPostgresqlPgstatementsAgent != nil:
		require.NotNil(t, res.Payload.QANPostgresqlPgstatementsAgent)
		agentID = res.Payload.QANPostgresqlPgstatementsAgent.AgentID
	case body.QANPostgresqlPgstatmonitorAgent != nil:
		require.NotNil(t, res.Payload.QANPostgresqlPgstatmonitorAgent)
		agentID = res.Payload.QANPostgresqlPgstatmonitorAgent.AgentID
	case body.RDSExporter != nil:
		require.NotNil(t, res.Payload.RDSExporter)
		agentID = res.Payload.RDSExporter.AgentID
	}
	require.NotEmpty(t, agentID)
	t.Cleanup(func() {
		RemoveAgents(t, agentID)
	})
	return res.Payload
}

// AddService adds a service with the specified body.
// The service will be automatically removed after the test.
func AddService(t *testing.T, body services.AddServiceBody) *services.AddServiceOKBody {
	t.Helper()

	params := &services.AddServiceParams{
		Body:    body,
		Context: t.Context(),
	}

	res, err := client.Default.ServicesService.AddService(params)
	require.NoError(t, err)
	require.NotNil(t, res)
	require.NotNil(t, res.Payload)

	var serviceID string
	switch {
	case body.External != nil:
		require.NotNil(t, res.Payload.External)
		serviceID = res.Payload.External.ServiceID
	case body.Mongodb != nil:
		require.NotNil(t, res.Payload.Mongodb)
		serviceID = res.Payload.Mongodb.ServiceID
	case body.Haproxy != nil:
		require.NotNil(t, res.Payload.Haproxy)
		serviceID = res.Payload.Haproxy.ServiceID
	case body.Mysql != nil:
		require.NotNil(t, res.Payload.Mysql)
		serviceID = res.Payload.Mysql.ServiceID
	case body.Postgresql != nil:
		require.NotNil(t, res.Payload.Postgresql)
		serviceID = res.Payload.Postgresql.ServiceID
	case body.Proxysql != nil:
		require.NotNil(t, res.Payload.Proxysql)
		serviceID = res.Payload.Proxysql.ServiceID
	case body.Valkey != nil:
		require.NotNil(t, res.Payload.Valkey)
		serviceID = res.Payload.Valkey.ServiceID
	}
	require.NotEmpty(t, serviceID)
	t.Cleanup(func() {
		RemoveServices(t, serviceID)
	})
	return res.Payload
}

var (
	gClient *grafanaclient.GrafanaHTTPAPI
	gOnce   sync.Once
)

// WaitServerReady checks if the server is ready by calling the readiness endpoint and fetching user details.
func WaitServerReady(ctx context.Context) error {
	return retryWithBackoff(ctx, 10, func() error {
		_, err := serverClient.Default.ServerService.Readiness(&server.ReadinessParams{
			Context: ctx,
		})
		if err != nil {
			return fmt.Errorf("failed to pass the server readiness probe: %w", err)
		}

		_, err = userClient.Default.UserService.GetUser(&user_service.GetUserParams{
			Context: ctx,
		})
		if err != nil {
			return fmt.Errorf("failed to get user details: %w", err)
		}
		return nil
	})
}

// retryWithBackoff retries fn with capped exponential backoff until it succeeds,
// attempts are exhausted, or ctx is done.
func retryWithBackoff(ctx context.Context, attempts int, fn func() error) error {
	var lastErr error
	for i := range attempts {
		lastErr = fn()
		if lastErr == nil {
			return nil
		}
		if i == attempts-1 {
			break
		}
		select {
		case <-time.After(backoff(i)):
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	return fmt.Errorf("retries exhausted: %w", lastErr)
}

func backoff(attempt int) time.Duration {
	d := time.Duration(1<<attempt) * time.Second
	return min(d, 5*time.Second)
}

// GetGrafanaClient creates and returns a Grafana client.
func GetGrafanaClient(t *testing.T) *grafanaclient.GrafanaHTTPAPI {
	t.Helper()

	gOnce.Do(func() {
		gURL := *BaseURL
		gURL.Path = "/graph/api"
		gClient = grafanaclient.New(Transport(&gURL, ServerInsecureTLS), grafanaclient.DefaultTransportConfig(), nil)
	})
	return gClient
}
