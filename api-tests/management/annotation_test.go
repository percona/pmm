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

package management

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"

	pmmapitests "github.com/percona/pmm/api-tests"
	inventoryClient "github.com/percona/pmm/api/inventorypb/json/client"
	"github.com/percona/pmm/api/inventorypb/json/client/nodes"
	"github.com/percona/pmm/api/inventorypb/json/client/services"
	"github.com/percona/pmm/api/managementpb/json/client"
	"github.com/percona/pmm/api/managementpb/json/client/annotation"
)

func TestAddAnnotation(t *testing.T) {
	t.Run("Add Basic Annotation", func(t *testing.T) {
		params := &annotation.AddAnnotationParams{
			Body: annotation.AddAnnotationBody{
				Text: "Annotation Text",
				Tags: []string{"tag1", "tag2"},
			},
			Context: pmmapitests.Context,
		}
		_, err := client.Default.Annotation.AddAnnotation(params)
		require.NoError(t, err)
	})

	t.Run("Add Empty Annotation", func(t *testing.T) {
		params := &annotation.AddAnnotationParams{
			Body: annotation.AddAnnotationBody{
				Text: "",
				Tags: []string{},
			},
			Context: pmmapitests.Context,
		}
		_, err := client.Default.Annotation.AddAnnotation(params)
		pmmapitests.AssertAPIErrorf(t, err, 400, codes.InvalidArgument, "invalid AddAnnotationRequest.Text: value length must be at least 1 runes")
	})

	t.Run("Non-existing service", func(t *testing.T) {
		params := &annotation.AddAnnotationParams{
			Body: annotation.AddAnnotationBody{
				Text:         "Some text",
				ServiceNames: []string{"no-service"},
			},
			Context: pmmapitests.Context,
		}
		_, err := client.Default.Annotation.AddAnnotation(params)
		pmmapitests.AssertAPIErrorf(t, err, 404, codes.NotFound, `Service with name "no-service" not found.`)
	})

	t.Run("Non-existing node", func(t *testing.T) {
		params := &annotation.AddAnnotationParams{
			Body: annotation.AddAnnotationBody{
				Text:     "Some text",
				NodeName: "no-node",
			},
			Context: pmmapitests.Context,
		}
		_, err := client.Default.Annotation.AddAnnotation(params)
		pmmapitests.AssertAPIErrorf(t, err, 404, codes.NotFound, `Node with name "no-node" not found.`)
	})

	t.Run("Existing service", func(t *testing.T) {
		nodeName := pmmapitests.TestString(t, "annotation-node")
		paramsNode := &nodes.AddNodeParams{
			Body: nodes.AddNodeBody{
				Generic: &nodes.AddNodeParamsBodyGeneric{
					NodeName: nodeName,
					Address:  "10.0.0.1",
				},
			},
			Context: pmmapitests.Context,
		}
		resNode, err := inventoryClient.Default.Nodes.AddNode(paramsNode)
		assert.NoError(t, err)
		genericNodeID := resNode.Payload.Generic.NodeID
		defer pmmapitests.RemoveNodes(t, genericNodeID)

		serviceName := pmmapitests.TestString(t, "annotation-service")
		paramsService := &services.AddMySQLServiceParams{
			Body: services.AddMySQLServiceBody{
				NodeID:      genericNodeID,
				Address:     "localhost",
				Port:        3306,
				ServiceName: serviceName,
			},
			Context: pmmapitests.Context,
		}
		resService, err := inventoryClient.Default.Services.AddMySQLService(paramsService)
		assert.NoError(t, err)
		require.NotNil(t, resService)
		serviceID := resService.Payload.Mysql.ServiceID
		defer pmmapitests.RemoveServices(t, serviceID)

		paramsAdd := &annotation.AddAnnotationParams{
			Body: annotation.AddAnnotationBody{
				Text:         "Some text",
				ServiceNames: []string{serviceName},
			},
			Context: pmmapitests.Context,
		}
		_, err = client.Default.Annotation.AddAnnotation(paramsAdd)
		require.NoError(t, err)
	})

	t.Run("Existing node", func(t *testing.T) {
		nodeName := pmmapitests.TestString(t, "annotation-node")
		params := &nodes.AddNodeParams{
			Body: nodes.AddNodeBody{
				Generic: &nodes.AddNodeParamsBodyGeneric{
					NodeName: nodeName,
					Address:  "10.0.0.1",
				},
			},
			Context: pmmapitests.Context,
		}
		res, err := inventoryClient.Default.Nodes.AddNode(params)
		assert.NoError(t, err)
		defer pmmapitests.RemoveNodes(t, res.Payload.Generic.NodeID)

		paramsAdd := &annotation.AddAnnotationParams{
			Body: annotation.AddAnnotationBody{
				Text:     "Some text",
				NodeName: nodeName,
			},
			Context: pmmapitests.Context,
		}
		_, err = client.Default.Annotation.AddAnnotation(paramsAdd)
		require.NoError(t, err)
	})
}
