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

// Package logship receives client and database logs streamed from PMM Clients over the existing agent
// channel and forwards them to the local OpenTelemetry Collector (OTLP/HTTP) for storage in ClickHouse.
package logship

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gopkg.in/reform.v1"

	agentv1 "github.com/percona/pmm/api/agent/v1"
	logshipv1 "github.com/percona/pmm/api/logship/v1"
	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/utils/logger"
)

// DefaultCollectorLogsEndpoint is the local OpenTelemetry Collector OTLP/HTTP logs endpoint.
const DefaultCollectorLogsEndpoint = "http://127.0.0.1:4318/v1/logs"

const exportTimeout = 10 * time.Second

// Service implements the LogShipService gRPC server.
type Service struct {
	db       *reform.DB
	endpoint string
	client   *http.Client
	l        *logrus.Entry

	logshipv1.UnimplementedLogShipServiceServer
}

// New creates a new log-shipping service forwarding to the given OTLP/HTTP logs endpoint.
func New(db *reform.DB, endpoint string) *Service {
	return &Service{
		db:       db,
		endpoint: endpoint,
		client:   &http.Client{Timeout: exportTimeout},
		l:        logrus.WithField("component", "logship"),
	}
}

// Ship handles the incoming stream of client/database log records (gRPC handler).
func (s *Service) Ship(stream grpc.ClientStreamingServer[logshipv1.ShipRequest, logshipv1.ShipResponse]) error {
	streamCtx := stream.Context()
	l := logger.Get(streamCtx)

	agentMD, err := agentv1.ReceiveAgentConnectMetadata(stream)
	if err != nil {
		l.Warnf("Disconnecting client: authentication failed: %v", err)
		return status.Error(codes.Unauthenticated, "Failed to receive agent metadata")
	}

	agent, err := models.FindAgentByID(s.db.Querier, agentMD.ID)
	if err != nil {
		l.Warnf("Disconnecting client: agent validation failed: %v", err)
		return status.Error(codes.InvalidArgument, "Invalid Agent ID: "+agentMD.ID)
	}
	if agent.AgentType != models.PMMAgentType {
		return status.Errorf(codes.InvalidArgument, "Agent with ID %s is not a pmm-agent", agentMD.ID)
	}

	for {
		select {
		case <-streamCtx.Done():
			return status.Error(codes.Canceled, "client disconnected")
		default:
		}

		msg, err := stream.Recv()
		if err != nil {
			if errors.Is(err, io.EOF) {
				return stream.SendAndClose(&logshipv1.ShipResponse{})
			}
			return err
		}

		if len(msg.Records) == 0 {
			continue // health ping or empty batch
		}

		// Log shipping is best-effort: on a forwarding error we drop the batch and keep the stream open
		// rather than disconnecting the agent, since the collector outage is usually transient.
		if err := s.export(streamCtx, agentMD.ID, msg); err != nil {
			s.l.Warnf("Failed to forward %d log records to the collector: %s", len(msg.Records), err)
		}
	}
}

// export converts a ShipRequest to OTLP/HTTP JSON and posts it to the local collector.
func (s *Service) export(ctx context.Context, agentID string, msg *logshipv1.ShipRequest) error {
	resourceAttrs := make([]otlpKeyValue, 0, len(msg.ResourceAttributes)+2) //nolint:mnd
	resourceAttrs = append(resourceAttrs, stringAttr("service.name", msg.ServiceName), stringAttr("pmm.agent_id", agentID))
	for k, v := range msg.ResourceAttributes {
		resourceAttrs = append(resourceAttrs, stringAttr(k, v))
	}

	records := make([]otlpLogRecord, 0, len(msg.Records))
	for _, r := range msg.Records {
		rec := otlpLogRecord{
			SeverityText: r.SeverityText,
			Body:         otlpAnyValue{StringValue: r.Body},
		}
		if r.Time != nil {
			rec.TimeUnixNano = strconv.FormatInt(r.Time.AsTime().UnixNano(), 10)
		}
		for k, v := range r.Attributes {
			rec.Attributes = append(rec.Attributes, stringAttr(k, v))
		}
		records = append(records, rec)
	}

	payload := otlpLogsData{
		ResourceLogs: []otlpResourceLogs{{
			Resource:  otlpResource{Attributes: resourceAttrs},
			ScopeLogs: []otlpScopeLogs{{LogRecords: records}},
		}},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.endpoint, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close() //nolint:errcheck
	_, _ = io.Copy(io.Discard, resp.Body)

	if resp.StatusCode/100 != 2 { //nolint:mnd
		return errors.New("collector returned status " + resp.Status)
	}
	return nil
}

// Minimal OTLP/HTTP JSON encoding for logs (subset of opentelemetry.proto.collector.logs.v1).

type otlpLogsData struct {
	ResourceLogs []otlpResourceLogs `json:"resourceLogs"`
}

type otlpResourceLogs struct {
	Resource  otlpResource    `json:"resource"`
	ScopeLogs []otlpScopeLogs `json:"scopeLogs"`
}

type otlpResource struct {
	Attributes []otlpKeyValue `json:"attributes,omitempty"`
}

type otlpScopeLogs struct {
	LogRecords []otlpLogRecord `json:"logRecords"`
}

type otlpLogRecord struct {
	TimeUnixNano string         `json:"timeUnixNano,omitempty"`
	SeverityText string         `json:"severityText,omitempty"`
	Body         otlpAnyValue   `json:"body"`
	Attributes   []otlpKeyValue `json:"attributes,omitempty"`
}

type otlpKeyValue struct {
	Key   string       `json:"key"`
	Value otlpAnyValue `json:"value"`
}

type otlpAnyValue struct {
	StringValue string `json:"stringValue"`
}

func stringAttr(key, value string) otlpKeyValue {
	return otlpKeyValue{Key: key, Value: otlpAnyValue{StringValue: value}}
}

// check interface.
var _ logshipv1.LogShipServiceServer = (*Service)(nil)
