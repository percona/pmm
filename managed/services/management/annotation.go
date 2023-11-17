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
	"strings"
	"time"

	"gopkg.in/reform.v1"

	managementpb "github.com/percona/pmm/api/managementpb"
	"github.com/percona/pmm/managed/models"
)

// AnnotationService Annotation Service.
type AnnotationService struct {
	db            *reform.DB
	grafanaClient grafanaClient
}

// NewAnnotationService create new Annotation Service.
func NewAnnotationService(db *reform.DB, grafanaClient grafanaClient) *AnnotationService {
	return &AnnotationService{
		db:            db,
		grafanaClient: grafanaClient,
	}
}

// AddAnnotation create annotation in grafana.
//
//nolint:unparam
func (as *AnnotationService) AddAnnotation(
	ctx context.Context,
	authorizationHeaders []string,
	req *managementpb.AddAnnotationRequest,
) (*managementpb.AddAnnotationResponse, error) {
	tags := req.Tags
	if len(req.ServiceNames) == 0 && req.NodeName == "" {
		tags = append([]string{"pmm_annotation"}, tags...)
	}
	var postfix []string
	if len(req.ServiceNames) != 0 {
		for _, sn := range req.ServiceNames {
			_, err := models.FindServiceByName(as.db.Querier, sn)
			if err != nil {
				return nil, err
			}
		}

		tags = append(tags, req.ServiceNames...)
		postfix = append(postfix, "Service Name: "+strings.Join(req.ServiceNames, ", "))
	}

	if req.NodeName != "" {
		_, err := models.FindNodeByName(as.db.Querier, req.NodeName)
		if err != nil {
			return nil, err
		}

		tags = append(tags, req.NodeName)
		postfix = append(postfix, "Node Name: "+req.NodeName)
	}

	if len(postfix) != 0 {
		req.Text += " (" + strings.Join(postfix, ". ") + ")"
	}

	_, err := as.grafanaClient.CreateAnnotation(ctx, tags, time.Now(), req.Text, authorizationHeaders[0])
	if err != nil {
		return nil, err
	}

	return &managementpb.AddAnnotationResponse{}, nil
}
