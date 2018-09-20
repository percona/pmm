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

package handlers

import (
	"context"

	"github.com/percona/pmm-managed/api"
	"github.com/percona/pmm-managed/services/grafana"
)

type AnnotationsServer struct {
	Grafana *grafana.Client
}

// Create creates annotation with given text and tags ("pmm_annotation" is added automatically).
func (s *AnnotationsServer) Create(ctx context.Context, req *api.AnnotationsCreateRequest) (*api.AnnotationsCreateResponse, error) {
	msg, err := s.Grafana.CreateAnnotation(ctx, req.Tags, req.Text)
	if err != nil {
		return nil, err
	}
	return &api.AnnotationsCreateResponse{
		Message: msg,
	}, nil
}

// check interfaces
var (
	_ api.AnnotationsServer = (*AnnotationsServer)(nil)
)
