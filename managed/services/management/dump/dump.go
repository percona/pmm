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

// Package dump exposes PMM Dump API.
package dump

import (
	"context"
	"fmt"
	"time"

	"github.com/AlekSi/pointer"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gopkg.in/reform.v1"

	dumpv1beta1 "github.com/percona/pmm/api/managementpb/dump"
	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/managed/services/dump"
	"github.com/percona/pmm/managed/services/grafana"
)

type Service struct {
	db *reform.DB
	l  *logrus.Entry

	dumpService   dumpService
	grafanaClient *grafana.Client

	dumpv1beta1.UnimplementedDumpsServer

	// TODO this service needs method for uploading dump artifacts via FTP
}

func New(db *reform.DB, grafanaClient *grafana.Client, dumpService dumpService) *Service {
	return &Service{
		db:            db,
		dumpService:   dumpService,
		grafanaClient: grafanaClient,
		l:             logrus.WithField("component", "management/dump"),
	}
}

func (s *Service) StartDump(ctx context.Context, req *dumpv1beta1.StartDumpRequest) (*dumpv1beta1.StartDumpResponse, error) {
	// TODO validate request

	apiKeyName := fmt.Sprintf("pmm-dump-%s", time.Now().Format(time.RFC3339))
	apiKeyID, apiKey, err := s.grafanaClient.CreateAdminAPIKey(ctx, apiKeyName)
	if err != nil {
		return nil, errors.Wrap(err, "cannot create Grafana admin API key")
	}

	defer func() {
		if err := s.grafanaClient.DeleteAPIKeyByID(ctx, apiKeyID); err != nil {
			s.l.Warnf("Failed to remove API key token after pmm dump completion: %+v", err)
		}
	}()
	dumpID, err := s.dumpService.StartDump(&dump.Params{
		APIKey:     apiKey,
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

func (s *Service) ListDumps(_ context.Context, req *dumpv1beta1.ListDumpsRequest) (*dumpv1beta1.ListDumpsResponse, error) {
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

func (s *Service) GetDumpLogs(_ context.Context, req *dumpv1beta1.GetLogsRequest) (*dumpv1beta1.GetLogsResponse, error) {
	filter := models.DumpLogsFilter{
		DumpID: req.DumpId,
		Offset: int(req.Offset),
	}
	if req.Limit > 0 {
		filter.Limit = pointer.ToInt(int(req.Limit))
	}

	dumpLogs, err := models.FindDumpLogs(s.db.Querier, filter)
	if err != nil {
		return nil, err
	}

	res := &dumpv1beta1.GetLogsResponse{
		Logs: make([]*dumpv1beta1.LogChunk, 0, len(dumpLogs)),
	}
	for _, log := range dumpLogs {
		if log.LastChunk {
			res.End = true
			break
		}
		res.Logs = append(res.Logs, &dumpv1beta1.LogChunk{
			ChunkId: log.ChunkID,
			Data:    log.Data,
		})
	}

	return res, nil
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
