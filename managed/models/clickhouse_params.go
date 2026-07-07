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

package models

import (
	"errors"
	"fmt"
	"net"
	"net/url"
	"strconv"
)

// ClickHouseParams represents ClickHouse server params.
type ClickHouseParams struct {
	url *url.URL
}

// ExternalClickHouse returns true if ClickHouse is configured externally.
func (p *ClickHouseParams) ExternalClickHouse() bool {
	return !internalAddr(p.url.Hostname())
}

// URL returns the ClickHouse URL.
func (p *ClickHouseParams) URL() *url.URL {
	u := *p.url
	return &u
}

// NewClickHouseParams returns validated ClickHouse configuration params,
// or an error if any required field is missing or malformed.
func NewClickHouseParams(addr, dbName, dbUsername, dbPassword string) (*ClickHouseParams, error) {
	if addr == "" {
		return nil, errors.New("addr is required")
	}
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return nil, fmt.Errorf("invalid addr %q: %w", addr, err)
	}
	if host == "" {
		return nil, fmt.Errorf("invalid addr %q: empty host", addr)
	}
	_, err = strconv.ParseUint(port, 10, 16)
	if err != nil {
		return nil, fmt.Errorf("invalid port in addr %q: %w", addr, err)
	}
	if dbName == "" {
		return nil, errors.New("database name is required")
	}
	if dbUsername == "" {
		return nil, errors.New("username is required")
	}

	return &ClickHouseParams{
		url: &url.URL{
			Scheme: "tcp",
			User:   url.UserPassword(dbUsername, dbPassword),
			Host:   addr,
			Path:   dbName,
		},
	}, nil
}
