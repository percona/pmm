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
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"

	pmmapitests "github.com/percona/pmm/api-tests"
	inventoryClient "github.com/percona/pmm/api/inventory/v1/json/client"
	nodes "github.com/percona/pmm/api/inventory/v1/json/client/nodes_service"
	services "github.com/percona/pmm/api/inventory/v1/json/client/services_service"
	"github.com/percona/pmm/api/management/v1/json/client"
	mservice "github.com/percona/pmm/api/management/v1/json/client/management_service"
)

// annotationSeriesTimeout is how long VictoriaMetrics may take to make a written series queryable.
const annotationSeriesTimeout = 60 * time.Second

func TestAddAnnotation(t *testing.T) {
	t.Parallel()

	t.Run("Add Basic Annotation", func(t *testing.T) {
		t.Parallel()

		params := &mservice.AddAnnotationParams{
			Body: mservice.AddAnnotationBody{
				Text: "Annotation Text",
				Tags: []string{"tag1", "tag2"},
			},
			Context: pmmapitests.Context,
		}
		_, err := client.Default.ManagementService.AddAnnotation(params)
		require.NoError(t, err)
	})

	t.Run("Add Empty Annotation", func(t *testing.T) {
		t.Parallel()

		params := &mservice.AddAnnotationParams{
			Body: mservice.AddAnnotationBody{
				Text: "",
				Tags: []string{},
			},
			Context: pmmapitests.Context,
		}
		_, err := client.Default.ManagementService.AddAnnotation(params)
		pmmapitests.AssertAPIErrorf(t, err, 400, codes.InvalidArgument, "invalid AddAnnotationRequest.Text: value length must be at least 1 runes")
	})

	t.Run("Non-existent service", func(t *testing.T) {
		t.Parallel()

		params := &mservice.AddAnnotationParams{
			Body: mservice.AddAnnotationBody{
				Text:         "Some text",
				ServiceNames: []string{"no-service"},
			},
			Context: pmmapitests.Context,
		}
		_, err := client.Default.ManagementService.AddAnnotation(params)
		pmmapitests.AssertAPIErrorf(t, err, 404, codes.NotFound, `Service with name "no-service" not found.`)
	})

	t.Run("Non-existent node", func(t *testing.T) {
		t.Parallel()

		params := &mservice.AddAnnotationParams{
			Body: mservice.AddAnnotationBody{
				Text:     "Some text",
				NodeName: "no-node",
			},
			Context: pmmapitests.Context,
		}
		_, err := client.Default.ManagementService.AddAnnotation(params)
		pmmapitests.AssertAPIErrorf(t, err, 404, codes.NotFound, `Node with name "no-node" not found.`)
	})

	t.Run("Tag length exceeded", func(t *testing.T) {
		t.Parallel()

		nodeName := pmmapitests.TestString(t, strings.Repeat("long-annotation-node-name-", 10))
		paramsNode := &nodes.AddNodeParams{
			Body: nodes.AddNodeBody{
				Generic: &nodes.AddNodeParamsBodyGeneric{
					NodeName: nodeName,
					Address:  pmmapitests.TestString(t, "10.0.0.1"),
				},
			},
			Context: pmmapitests.Context,
		}
		resNode, err := inventoryClient.Default.NodesService.AddNode(paramsNode)
		require.NoError(t, err)
		genericNodeID := resNode.Payload.Generic.NodeID
		t.Cleanup(func() {
			pmmapitests.RemoveNodes(t, genericNodeID)
		})

		params := &mservice.AddAnnotationParams{
			Body: mservice.AddAnnotationBody{
				Text:     "Some text",
				NodeName: nodeName,
			},
			Context: pmmapitests.Context,
		}
		_, err = client.Default.ManagementService.AddAnnotation(params)
		pmmapitests.AssertAPIErrorf(t, err, 400, codes.InvalidArgument, "tag length cannot exceed 100 characters, tag: "+nodeName)
	})

	t.Run("Existing service", func(t *testing.T) {
		t.Parallel()

		nodeName := pmmapitests.TestString(t, "annotation-node")
		paramsNode := &nodes.AddNodeParams{
			Body: nodes.AddNodeBody{
				Generic: &nodes.AddNodeParamsBodyGeneric{
					NodeName: nodeName,
					Address:  pmmapitests.TestString(t, "10.0.0.1"),
				},
			},
			Context: pmmapitests.Context,
		}
		resNode, err := inventoryClient.Default.NodesService.AddNode(paramsNode)
		require.NoError(t, err)
		genericNodeID := resNode.Payload.Generic.NodeID
		t.Cleanup(func() {
			pmmapitests.RemoveNodes(t, genericNodeID)
		})

		serviceName := "annotation-service"
		paramsService := &services.AddServiceParams{
			Body: services.AddServiceBody{
				Mysql: &services.AddServiceParamsBodyMysql{
					NodeID:      genericNodeID,
					Address:     pmmapitests.TestString(t, "localhost"),
					Port:        3306,
					ServiceName: serviceName,
					Environment: "annotation-env",
				},
			},
			Context: pmmapitests.Context,
		}

		resService, err := inventoryClient.Default.ServicesService.AddService(paramsService)
		require.NoError(t, err)
		require.NotNil(t, resService)
		serviceID := resService.Payload.Mysql.ServiceID
		t.Cleanup(func() {
			pmmapitests.RemoveServices(t, serviceID)
		})

		annotationID := addAnnotation(t, mservice.AddAnnotationBody{
			Text:         "Some text",
			ServiceNames: []string{serviceName},
		})

		series := waitForAnnotationSeries(t, annotationID)
		require.Len(t, series, 1)
		assert.Equal(t, serviceName, series[0]["service_name"])
		assert.Equal(t, nodeName, series[0]["node_name"])
		assert.Equal(t, "annotation-env", series[0]["environment"])
	})

	t.Run("Existing node", func(t *testing.T) {
		t.Parallel()

		nodeName := "annotation-node"
		params := &nodes.AddNodeParams{
			Body: nodes.AddNodeBody{
				Generic: &nodes.AddNodeParamsBodyGeneric{
					NodeName: nodeName,
					Address:  pmmapitests.TestString(t, "10.0.0.1"),
				},
			},
			Context: pmmapitests.Context,
		}
		res, err := inventoryClient.Default.NodesService.AddNode(params)
		require.NoError(t, err)
		nodeID := res.Payload.Generic.NodeID
		t.Cleanup(func() {
			pmmapitests.RemoveNodes(t, nodeID)
		})

		annotationID := addAnnotation(t, mservice.AddAnnotationBody{
			Text:     "Some text",
			NodeName: nodeName,
		})

		series := waitForAnnotationSeries(t, annotationID)
		require.Len(t, series, 1)
		assert.Equal(t, nodeName, series[0]["node_name"])
		assert.Empty(t, series[0]["service_name"])
	})

	t.Run("Too many labels", func(t *testing.T) {
		t.Parallel()

		const customLabelsCount = 39
		customLabels := make(map[string]string, customLabelsCount)
		for i := range customLabelsCount {
			customLabels[fmt.Sprintf("label_%02d", i)] = "value"
		}

		nodeName := pmmapitests.TestString(t, "annotation-node")
		pmmapitests.AddNode(t, &nodes.AddNodeBody{
			Remote: &nodes.AddNodeParamsBodyRemote{
				NodeName:     nodeName,
				Address:      pmmapitests.TestString(t, "10.0.0.1"),
				CustomLabels: customLabels,
			},
		})

		params := &mservice.AddAnnotationParams{
			Body: mservice.AddAnnotationBody{
				Text:     "Some text",
				NodeName: nodeName,
			},
			Context: pmmapitests.Context,
		}
		_, err := client.Default.ManagementService.AddAnnotation(params)
		pmmapitests.AssertAPIErrorf(t, err, 400, codes.InvalidArgument, "Cannot annotate node '"+nodeName+"': too many labels")
	})
}

func TestRemoveAnnotation(t *testing.T) {
	t.Parallel()

	t.Run("Existing annotation", func(t *testing.T) {
		t.Parallel()

		node := pmmapitests.AddGenericNode(t, pmmapitests.TestString(t, "annotation-node"))
		annotationID := addAnnotation(t, mservice.AddAnnotationBody{
			Text:     "Some text",
			NodeName: node.NodeName,
		})
		require.Len(t, waitForAnnotationSeries(t, annotationID), 1)

		params := &mservice.RemoveAnnotationParams{
			AnnotationID: annotationID,
			Context:      pmmapitests.Context,
		}
		res, err := client.Default.ManagementService.RemoveAnnotation(params)
		require.NoError(t, err)
		assert.True(t, res.IsCode(http.StatusOK))
		assert.Empty(t, annotationSeries(t, annotationID))
	})

	t.Run("Non-existent annotation", func(t *testing.T) {
		t.Parallel()

		annotationID := strconv.FormatInt(math.MaxInt64, 10)
		params := &mservice.RemoveAnnotationParams{
			AnnotationID: annotationID,
			Context:      pmmapitests.Context,
		}
		_, err := client.Default.ManagementService.RemoveAnnotation(params)
		pmmapitests.AssertAPIErrorf(t, err, 404, codes.NotFound, "Annotation with ID "+annotationID+" not found.")
	})
}

// addAnnotation adds an annotation and removes it after the test, unless the test has removed it already.
func addAnnotation(t *testing.T, body mservice.AddAnnotationBody) string {
	t.Helper()

	res, err := client.Default.ManagementService.AddAnnotation(&mservice.AddAnnotationParams{
		Body:    body,
		Context: pmmapitests.Context,
	})
	require.NoError(t, err)
	require.NotNil(t, res.Payload)
	require.NotEmpty(t, res.Payload.AnnotationID)

	annotationID := res.Payload.AnnotationID
	t.Cleanup(func() {
		_, _ = client.Default.ManagementService.RemoveAnnotation(&mservice.RemoveAnnotationParams{
			AnnotationID: annotationID,
			Context:      pmmapitests.Context,
		})
	})

	return annotationID
}

// annotationSeries returns the labels of the annotation's pmm_annotation_event series, as VictoriaMetrics reports them now.
func annotationSeries(t *testing.T, annotationID string) []map[string]string {
	t.Helper()

	u := *pmmapitests.BaseURL
	u.Path = "/prometheus/api/v1/query"
	u.RawQuery = url.Values{"query": {fmt.Sprintf(`pmm_annotation_event{pmm_annotation_id="%s"}`, annotationID)}}.Encode()

	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, u.String(), nil)
	require.NoError(t, err)

	resp, err := pmmapitests.HTTPClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode, "unexpected response: %s", body)

	var res struct {
		Status string `json:"status"`
		Data   struct {
			Result []struct {
				Metric map[string]string `json:"metric"`
			} `json:"result"`
		} `json:"data"`
	}
	err = json.Unmarshal(body, &res)
	require.NoError(t, err)
	require.Equal(t, "success", res.Status, "unexpected response: %s", body)

	series := make([]map[string]string, 0, len(res.Data.Result))
	for _, r := range res.Data.Result {
		series = append(series, r.Metric)
	}

	return series
}

// waitForAnnotationSeries polls VictoriaMetrics until the series of the annotation is queryable.
func waitForAnnotationSeries(t *testing.T, annotationID string) []map[string]string {
	t.Helper()

	deadline := time.Now().Add(annotationSeriesTimeout)
	for {
		series := annotationSeries(t, annotationID)
		if len(series) != 0 {
			return series
		}

		require.True(t, time.Now().Before(deadline), "no series of annotation %s within %s", annotationID, annotationSeriesTimeout)
		time.Sleep(time.Second)
	}
}
