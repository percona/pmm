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

// Package grpc provides gRPC servers.
package grpc

import (
	"context"
	"fmt"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"gopkg.in/reform.v1"

	"github.com/percona/pmm/api/managementpb"
	"github.com/percona/pmm/managed/services/grafana"
	"github.com/percona/pmm/managed/services/management"
)

// AnnotationServer is a server for making annotations in Grafana.
type AnnotationServer struct {
	svc *management.AnnotationService

	managementpb.UnimplementedAnnotationServer
}

// NewAnnotationServer creates Annotation Server.
func NewAnnotationServer(db *reform.DB, grafanaClient *grafana.Client) *AnnotationServer {
	return &AnnotationServer{
		svc: management.NewAnnotationService(db, grafanaClient),
	}
}

// AddAnnotation adds annotation to Grafana.
func (as *AnnotationServer) AddAnnotation(ctx context.Context, req *managementpb.AddAnnotationRequest) (*managementpb.AddAnnotationResponse, error) {
	headers, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, fmt.Errorf("cannot get headers from metadata")
	}
	// get authorization from headers.
	authorizationHeaders := headers.Get("Authorization")
	if len(authorizationHeaders) == 0 {
		return nil, status.Error(codes.Unauthenticated, "Authorization error.")
	}

	return as.svc.AddAnnotation(ctx, authorizationHeaders, req)
}
