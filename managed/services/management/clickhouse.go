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
	"context"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"time"

	"github.com/AlekSi/pointer"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gopkg.in/reform.v1"

	inventoryv1 "github.com/percona/pmm/api/inventory/v1"
	managementv1 "github.com/percona/pmm/api/management/v1"
	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/managed/services"
)

// defaultClickHouseNativeMetricsPort is the default port of the ClickHouse
// native Prometheus endpoint (the <prometheus> server config section).
const defaultClickHouseNativeMetricsPort = 9363

// clickHouseNativeProbeTimeout bounds the auto-probe of the native endpoint.
const clickHouseNativeProbeTimeout = 3 * time.Second

// probeClickHouseNativeEndpoint reports whether the ClickHouse native
// Prometheus endpoint answers an HTTP GET on {address}:{port}/metrics.
func probeClickHouseNativeEndpoint(ctx context.Context, address string, port uint16) bool {
	if address == "" {
		return false
	}

	probeCtx, cancel := context.WithTimeout(ctx, clickHouseNativeProbeTimeout)
	defer cancel()

	url := fmt.Sprintf("http://%s/metrics", net.JoinHostPort(address, strconv.Itoa(int(port))))
	req, err := http.NewRequestWithContext(probeCtx, http.MethodGet, url, nil)
	if err != nil {
		return false
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close() //nolint:errcheck

	return resp.StatusCode == http.StatusOK
}

// addClickHouse adds a new ClickHouse service. Its metrics reach PMM either
// through the ClickHouse native Prometheus endpoint (modelled as an external
// exporter) or through a PMM-managed clickhouse_exporter, selected by
// req.MetricsSource (auto-probe when unspecified).
func (s *ManagementService) addClickHouse(ctx context.Context, req *managementv1.AddClickHouseServiceParams) (*managementv1.AddServiceResponse, error) {
	clickhouse := &managementv1.ClickHouseServiceResult{}

	nativePort := uint16(req.NativeMetricsPort) //nolint:gosec
	if nativePort == 0 {
		nativePort = defaultClickHouseNativeMetricsPort
	}

	errTx := s.db.InTransactionContext(ctx, nil, func(tx *reform.TX) error {
		nodeID, err := nodeID(tx, req.NodeId, req.NodeName, req.AddNode, req.Address)
		if err != nil {
			return err
		}

		service, err := models.AddNewService(tx.Querier, models.ClickHouseServiceType, &models.AddDBMSServiceParams{
			ServiceName:    req.ServiceName,
			NodeID:         nodeID,
			Environment:    req.Environment,
			Cluster:        req.Cluster,
			ReplicationSet: req.ReplicationSet,
			Address:        pointer.ToStringOrNil(req.Address),
			Port:           pointer.ToUint16OrNil(uint16(req.Port)), //nolint:gosec
			Socket:         pointer.ToStringOrNil(req.Socket),
			CustomLabels:   req.CustomLabels,
		})
		if err != nil {
			return err
		}

		inventoryService, err := services.ToAPIService(service)
		if err != nil {
			return err
		}
		clickhouse.Service = inventoryService.(*inventoryv1.ClickHouseService) //nolint:forcetypeassert

		// Resolve the metrics source: auto-probe when unspecified; a forced
		// native source that fails the probe is a precondition failure.
		source := req.MetricsSource
		switch source {
		case managementv1.MetricsSource_METRICS_SOURCE_UNSPECIFIED:
			if probeClickHouseNativeEndpoint(ctx, req.Address, nativePort) {
				source = managementv1.MetricsSource_METRICS_SOURCE_NATIVE
			} else {
				source = managementv1.MetricsSource_METRICS_SOURCE_EXPORTER
			}
		case managementv1.MetricsSource_METRICS_SOURCE_NATIVE:
			if !probeClickHouseNativeEndpoint(ctx, req.Address, nativePort) {
				return status.Errorf(codes.FailedPrecondition,
					"ClickHouse native Prometheus endpoint is not reachable at %s:%d; "+
						"enable the <prometheus> server config section or use --metrics-source=exporter",
					req.Address, nativePort)
			}
		case managementv1.MetricsSource_METRICS_SOURCE_EXPORTER:
			// Explicit managed-exporter source; no native-endpoint probe needed.
		}

		if source == managementv1.MetricsSource_METRICS_SOURCE_NATIVE {
			row, err := models.CreateExternalExporter(tx.Querier, &models.CreateExternalExporterParams{
				RunsOnNodeID:  nodeID,
				ServiceID:     service.ServiceID,
				Username:      req.Username,
				Password:      req.Password,
				Scheme:        "http",
				MetricsPath:   "/metrics",
				ListenPort:    uint32(nativePort),
				CustomLabels:  req.CustomLabels,
				TLSSkipVerify: req.TlsSkipVerify,
			})
			if err != nil {
				return err
			}

			// Record that this service is monitored via the native endpoint.
			row.ClickHouseOptions = models.ClickHouseOptions{
				NativeEndpoint:    true,
				NativeMetricsPort: nativePort,
			}
			err = tx.Update(row)
			if err != nil {
				return err
			}

			agent, err := services.ToAPIAgent(tx.Querier, row)
			if err != nil {
				return err
			}
			clickhouse.ExternalExporter = agent.(*inventoryv1.ExternalExporter) //nolint:forcetypeassert
			return nil
		}

		req.MetricsMode, err = supportedMetricsMode(req.MetricsMode, req.PmmAgentId)
		if err != nil {
			return err
		}

		row, err := models.CreateAgent(tx.Querier, models.ClickHouseExporterType, &models.CreateAgentParams{
			PMMAgentID:    req.PmmAgentId,
			ServiceID:     service.ServiceID,
			Username:      req.Username,
			Password:      req.Password,
			TLS:           req.Tls,
			TLSSkipVerify: req.TlsSkipVerify,
			ExporterOptions: models.ExporterOptions{
				ExposeExporter: req.ExposeExporter,
				PushMetrics:    isPushMode(req.MetricsMode),
			},
			ClickHouseOptions: models.ClickHouseOptionsFromRequest(req),
		})
		if err != nil {
			return err
		}
		if !req.SkipConnectionCheck {
			err = s.cc.CheckConnectionToService(ctx, tx.Querier, service, row)
			if err != nil {
				return err
			}

			err = s.sib.GetInfoFromService(ctx, tx.Querier, service, row)
			if err != nil {
				return err
			}
		}

		agent, err := services.ToAPIAgent(tx.Querier, row)
		if err != nil {
			return err
		}
		clickhouse.ClickhouseExporter = agent.(*inventoryv1.ClickHouseExporter) //nolint:forcetypeassert
		return nil
	})
	if errTx != nil {
		return nil, errTx
	}

	s.state.RequestStateUpdate(ctx, req.PmmAgentId)
	res := &managementv1.AddServiceResponse{
		Service: &managementv1.AddServiceResponse_Clickhouse{
			Clickhouse: clickhouse,
		},
	}
	return res, nil
}
