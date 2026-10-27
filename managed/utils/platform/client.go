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

// Package platform implements HTTP client for anonymous telemetry.
package platform

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	telemetryv1 "github.com/percona/platform/gen/telemetry/generic"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"

	"github.com/percona/pmm/managed/utils/envvars"
	"github.com/percona/pmm/utils/tlsconfig"
)

// Client is HTTP client for sending anonymous telemetry.
type Client struct {
	address string
	l       *logrus.Entry
	client  http.Client
}

// NewClient creates new telemetry client.
func NewClient(address string) *Client {
	l := logrus.WithField("component", "telemetry client")

	tlsConfig := tlsconfig.Get()
	tlsConfig.InsecureSkipVerify = envvars.GetPlatformInsecure()

	return &Client{
		l:       l,
		address: address,
		client: http.Client{
			Timeout: envvars.GetPlatformAPITimeout(l),
			Transport: &http.Transport{
				TLSClientConfig: tlsConfig,
				// Go respects proxy configuration by default, setting a transport
				// without proxy would make the requests ignore proxy settings.
				Proxy: http.ProxyFromEnvironment,
			},
		},
	}
}

// SendTelemetry sends anonymous telemetry data to Percona Platform.
func (c *Client) SendTelemetry(ctx context.Context, report *telemetryv1.ReportRequest) error {
	const path = "/v1/telemetry/GenericReport"

	body, err := protojson.Marshal(report)
	if err != nil {
		return err
	}

	_, err = c.makeRequest(ctx, http.MethodPost, path, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to send telemetry data: %w", err)
	}

	return nil
}

// makeRequest makes an anonymous request to Percona Platform.
func (c *Client) makeRequest(ctx context.Context, method, path string, body io.Reader) ([]byte, error) {
	endpoint := c.address + path
	req, err := http.NewRequestWithContext(ctx, method, endpoint, body)
	if err != nil {
		return nil, err
	}

	req.Header.Add("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close() //nolint:gosec,errcheck,nolintlint

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		var gwErr struct {
			Message string `json:"message"`
			Code    uint32 `json:"code"`
		}

		err := json.Unmarshal(bodyBytes, &gwErr)
		if err != nil {
			c.l.Errorf("Failed to send telemetry and failed to decode error message: %s", err)
			return nil, err
		}
		return nil, status.Error(codes.Code(gwErr.Code), gwErr.Message)
	}

	return bodyBytes, nil
}
