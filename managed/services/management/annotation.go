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
	"strings"
	"time"

	"github.com/golang/protobuf/ptypes/empty"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	managementv1 "github.com/percona/pmm/api/management/v1"
	"github.com/percona/pmm/managed/models"
)

// AddAnnotation create annotation in grafana.
//
//nolint:unparam
func (s *ManagementService) AddAnnotation(ctx context.Context, req *managementv1.AddAnnotationRequest) (*empty.Empty, error) {
	headers, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, fmt.Errorf("cannot get headers from metadata")
	}
	// get authorization from headers.
	authorizationHeaders := headers.Get("Authorization")
	if len(authorizationHeaders) == 0 {
		return nil, status.Error(codes.Unauthenticated, "Authorization error.")
	}

	tags := req.Tags
	if len(req.ServiceNames) == 0 && req.NodeName == "" {
		tags = append([]string{"pmm_annotation"}, tags...)
	}
	var postfix []string
	if len(req.ServiceNames) != 0 {
		for _, sn := range req.ServiceNames {
			_, err := models.FindServiceByName(s.db.Querier, sn)
			if err != nil {
				return nil, err
			}
		}

		tags = append(tags, req.ServiceNames...)
		postfix = append(postfix, "Service Name: "+strings.Join(req.ServiceNames, ", "))
	}

	if req.NodeName != "" {
		_, err := models.FindNodeByName(s.db.Querier, req.NodeName)
		if err != nil {
			return nil, err
		}

		tags = append(tags, req.NodeName)
		postfix = append(postfix, "Node Name: "+req.NodeName)
	}

	if len(postfix) != 0 {
		req.Text += " (" + strings.Join(postfix, ". ") + ")"
	}

	_, err := s.grafanaClient.CreateAnnotation(ctx, tags, time.Now(), req.Text, authorizationHeaders[0])
	if err != nil {
		return nil, err
	}

	return &empty.Empty{}, nil
}
