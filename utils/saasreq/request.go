// pmm-managed
// Copyright (C) 2017 Percona LLC
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

// Package saasreq provides http/https connection setup for Percona Platform.
package saasreq

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"time"

	"github.com/percona/pmm/utils/tlsconfig"
)

const dialTimeout = 10 * time.Second

// MakeRequest creates http/https POST request to Percona Platform
func MakeRequest(ctx context.Context, method string, endpoint, accessToken string, body io.Reader) ([]byte, error) {
	u, err := url.Parse(endpoint)
	if err != nil {
		return nil, err
	}

	tlsConfig := tlsconfig.Get()
	tlsConfig.ServerName = u.Host

	ctx, cancel := context.WithTimeout(ctx, dialTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, method, endpoint, body)
	if err != nil {
		return nil, err
	}

	h := req.Header
	h.Add("Content-Type", "application/json")
	if accessToken != "" {
		h.Add("Authorization", fmt.Sprintf("Bearer %s", accessToken))
	}

	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: tlsConfig,
		},
	}
	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	defer res.Body.Close() //nolint:errcheck

	bodyBytes, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to dial %s, response body: %s", endpoint, bodyBytes)
	}

	return bodyBytes, nil
}
