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

// Package saasdial provides gRPC connection setup for Percona Platform.
package saasdial

import (
	"context"
	"net"
	"time"

	"github.com/percona/pmm/utils/tlsconfig"
	"github.com/percona/pmm/version"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

const dialTimeout = 10 * time.Second

// Dial creates gRPC connection to Percona Platform
func Dial(ctx context.Context, sessionID string, hostPort string) (*grpc.ClientConn, error) {
	host, _, err := net.SplitHostPort(hostPort)
	if err != nil {
		return nil, errors.Wrap(err, "failed to set percona platform host")
	}
	tlsConfig := tlsconfig.Get()
	tlsConfig.ServerName = host

	opts := []grpc.DialOption{
		// replacement is marked as experimental
		grpc.WithBackoffMaxDelay(dialTimeout), //nolint:staticcheck

		grpc.WithBlock(),
		grpc.WithUserAgent("pmm-managed/" + version.Version),
		grpc.WithTransportCredentials(credentials.NewTLS(tlsConfig)),
	}

	if sessionID != "" {
		opts = append(opts, grpc.WithPerRPCCredentials(&platformAuth{sessionID: sessionID}))
	}

	ctx, cancel := context.WithTimeout(ctx, dialTimeout)
	defer cancel()
	cc, err := grpc.DialContext(ctx, hostPort, opts...)
	if err != nil {
		return nil, errors.Wrap(err, "failed to dial")
	}

	return cc, nil
}
