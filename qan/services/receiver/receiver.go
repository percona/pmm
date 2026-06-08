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

// Package receiver implements the QAN CollectorService that ingests metrics buckets.
package receiver

import (
	"context"

	"github.com/sirupsen/logrus"

	qanv1 "github.com/percona/pmm/api/qan/v1"
	"github.com/percona/pmm/qan/models"
)

// Service implements the gRPC CollectorService.
type Service struct {
	ingestor *models.Ingestor
	l        *logrus.Entry

	qanv1.UnimplementedCollectorServiceServer
}

// NewService returns a CollectorService backed by the given ingestor.
func NewService(ingestor *models.Ingestor) *Service {
	return &Service{
		ingestor: ingestor,
		l:        logrus.WithField("component", "receiver"),
	}
}

// Collect stores metrics buckets received from pmm-agent (via pmm-managed).
func (s *Service) Collect(ctx context.Context, req *qanv1.CollectRequest) (*qanv1.CollectResponse, error) {
	err := s.ingestor.Save(ctx, req.MetricsBucket)
	if err != nil {
		s.l.Errorf("Failed to save %d bucket(s): %v", len(req.MetricsBucket), err)
		return nil, err
	}
	return &qanv1.CollectResponse{}, nil
}
