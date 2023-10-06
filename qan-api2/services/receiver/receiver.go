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

// Package receiver provides service for collecting QAN data.
package receiver

import (
	"context"

	"github.com/sirupsen/logrus"

	qanpb "github.com/percona/pmm/api/qanpb"
	"github.com/percona/pmm/qan-api2/models"
	"github.com/percona/pmm/qan-api2/utils/logger"
)

// Service implements gRPC service to communicate with agent.
type Service struct {
	mbm *models.MetricsBucket
	l   *logrus.Entry //nolint:unused

	qanpb.UnimplementedCollectorServer
}

// NewService create new insstance of Service.
func NewService(mbm *models.MetricsBucket) *Service {
	return &Service{
		mbm: mbm,
	}
}

// Collect implements rpc to store data collected from slowlog/perf schema etc.
func (s *Service) Collect(ctx context.Context, req *qanpb.CollectRequest) (*qanpb.CollectResponse, error) {
	logger.Get(ctx).Infof("Saving %d MetricsBucket.", len(req.MetricsBucket))

	if err := s.mbm.Save(req); err != nil {
		return nil, err
	}
	return &qanpb.CollectResponse{}, nil
}
