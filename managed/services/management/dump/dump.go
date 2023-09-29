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

package dump

import (
	"context"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gopkg.in/reform.v1"

	dumpv1beta1 "github.com/percona/pmm/api/managementpb/dump"
	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/managed/services/dump"
)

type Service struct {
	db *reform.DB
	l  *logrus.Entry

	dumpService dumpService

	dumpv1beta1.UnimplementedDumpsServer
}

func New(db *reform.DB, dumpService dumpService) *Service {
	return &Service{
		db:          db,
		dumpService: dumpService,
		l:           logrus.WithField("component", "management/dump"),
	}
}

func (s *Service) StartDump(ctx context.Context, req *dumpv1beta1.StartDumpRequest) (*dumpv1beta1.StartDumpResponse, error) {
	// TODO validate request

	dumpID, err := s.dumpService.StartDump(&dump.Params{
		StartTime:  req.StartTime.AsTime(),
		EndTime:    req.EndTime.AsTime(),
		ExportQAN:  req.ExportQan,
		IgnoreLoad: req.IgnoreLoad,
		// TODO handle node ids
	})
	if err != nil {
		return nil, err
	}

	return &dumpv1beta1.StartDumpResponse{DumpId: dumpID}, nil
}

func (s *Service) ListDumps(ctx context.Context, req *dumpv1beta1.ListDumpsRequest) (*dumpv1beta1.ListDumpsResponse, error) {
	dumps, err := models.FindDumps(s.db.Querier, models.DumpFilters{})
	if err != nil {
		return nil, err
	}

	dumpsResponse := make([]*dumpv1beta1.Dump, 0, len(dumps))
	for _, dump := range dumps {
		d, err := convertDump(dump)
		if err != nil {
			return nil, err
		}

		dumpsResponse = append(dumpsResponse, d)
	}

	return &dumpv1beta1.ListDumpsResponse{
		Dumps: dumpsResponse,
	}, nil
}

func (s *Service) DeleteDump(ctx context.Context, req *dumpv1beta1.DeleteDumpRequest) (*dumpv1beta1.DeleteDumpResponse, error) {
	// TODO implement me
	panic("implement me")
}

func (s *Service) GetDumpLogs(ctx context.Context, req *dumpv1beta1.GetDumpLogsRequest) (*dumpv1beta1.GetDumpLogsResponse, error) {
	// TODO implement me
	panic("implement me")
}

func convertDump(dump *models.Dump) (*dumpv1beta1.Dump, error) {
	status, err := convertDumpStatus(dump.Status)
	if err != nil {
		return nil, errors.Wrap(err, "failed to convert dump status")
	}

	return &dumpv1beta1.Dump{
		DumpId:    dump.ID,
		Status:    status,
		NodeIds:   dump.NodeIDs,
		StartTime: timestamppb.New(dump.StartTime),
		EndTime:   timestamppb.New(dump.EndTime),
		CreatedAt: timestamppb.New(dump.CreatedAt),
	}, nil
}

func convertDumpStatus(status models.DumpStatus) (dumpv1beta1.DumpStatus, error) {
	switch status {
	case models.DumpStatusSuccess:
		return dumpv1beta1.DumpStatus_BACKUP_STATUS_SUCCESS, nil
	case models.DumpStatusError:
		return dumpv1beta1.DumpStatus_BACKUP_STATUS_ERROR, nil
	case models.DumpStatusInProgress:
		return dumpv1beta1.DumpStatus_BACKUP_STATUS_IN_PROGRESS, nil
	default:
		return dumpv1beta1.DumpStatus_BACKUP_STATUS_INVALID, errors.Errorf("invalid status '%s'", status)
	}
}
