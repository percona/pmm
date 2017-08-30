// pmm-managed
// Copyright (C) 2017 Percona LLC
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published
// by the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program. If not, see <http://www.gnu.org/licenses/>.

package api

import (
	"google.golang.org/grpc"
)

// Client is a client for pmm-managed APIs.
type Client struct {
	BaseClient
	// TODO AlertsClient
	ScrapeJobsClient
}

// NewClient creates new Client for a given connection.
func NewClient(cc *grpc.ClientConn) *Client {
	return &Client{
		BaseClient: NewBaseClient(cc),
		// TODO AlertsClient
		ScrapeJobsClient: NewScrapeJobsClient(cc),
	}
}
