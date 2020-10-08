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

// Package dbaas contains business logic of working with dbaas-controller.
package dbaas

import (
	"context"
	"time"

	controllerv1beta1 "github.com/percona-platform/dbaas-api/gen/controller"
	"github.com/percona/pmm/version"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/backoff"
)

const (
	dialTimeout = 5 * time.Second
)

// Client represents dbaas-controller client to use dbaas services.
type Client struct {
	XtraDBClusterAPIClient XtraDBClusterAPIConnector
}

// NewClient returns new client for given gRPC connection.
func NewClient(ctx context.Context, address string) (*Client, error) {
	conn, err := dial(ctx, address)
	if err != nil {
		return nil, err
	}

	return &Client{
		XtraDBClusterAPIClient: controllerv1beta1.NewXtraDBClusterAPIClient(conn),
	}, nil
}

func dial(ctx context.Context, address string) (*grpc.ClientConn, error) {
	opts := []grpc.DialOption{
		grpc.WithInsecure(),
		grpc.WithConnectParams(grpc.ConnectParams{Backoff: backoff.Config{MaxDelay: 2 * time.Second}, MinConnectTimeout: 10 * time.Second}),
		grpc.WithUserAgent("pmm-managed/" + version.Version),
	}

	ctx, cancel := context.WithTimeout(ctx, dialTimeout)
	defer cancel()
	cc, err := grpc.DialContext(ctx, address, opts...)
	if err != nil {
		return nil, errors.Wrap(err, "failed to dial")
	}

	return cc, nil
}
