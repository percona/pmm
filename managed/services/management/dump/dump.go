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
	"bufio"
	"context"
	"encoding/base64"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/AlekSi/pointer"
	"github.com/pkg/errors"
	"github.com/pkg/sftp"
	"github.com/sirupsen/logrus"
	"golang.org/x/crypto/ssh"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gopkg.in/reform.v1"

	dumpv1beta1 "github.com/percona/pmm/api/dump/v1beta1"
	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/managed/services/dump"
	"github.com/percona/pmm/managed/services/grafana"
)

// Service represents a structure for managing dump-related operations.
type Service struct {
	db *reform.DB
	l  *logrus.Entry

	dumpService   dumpService
	grafanaClient *grafana.Client

	dumpv1beta1.UnimplementedDumpServiceServer
}

// New creates a new instance of the Service with the provided dependencies.
func New(db *reform.DB, grafanaClient *grafana.Client, dumpService dumpService) *Service {
	return &Service{
		db:            db,
		dumpService:   dumpService,
		grafanaClient: grafanaClient,
		l:             logrus.WithField("component", "management/dump"),
	}
}

// StartDump starts a dump based on the provided context and request.
func (s *Service) StartDump(ctx context.Context, req *dumpv1beta1.StartDumpRequest) (*dumpv1beta1.StartDumpResponse, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, errors.New("can't get request metadata")
	}

	// Here we're trying to extract authentication credentials from incoming request. We need to forward them to pmm-dump tool.
	authHeader, cookieHeader := md.Get("grpcgateway-authorization"), md.Get("grpcgateway-cookie")

	// pmm-dump supports user/pass authentication, API token or cookie.
	var token, cookie, user, password string
	if len(authHeader) != 0 {
		// If auth header type is `Basic`, try to extract the user and password.
		if basic, ok := strings.CutPrefix(authHeader[0], "Basic"); ok {
			decodedBasic, err := base64.StdEncoding.DecodeString(strings.TrimSpace(basic))
			if err != nil {
				return nil, errors.Wrap(err, "failed to decode basic authorization header")
			}

			s := strings.Split(string(decodedBasic), ":")
			if len(s) < 2 {
				return nil, errors.New("failed to parse basic authorization header")
			}
			user, password = s[0], s[1]
		}

		// If auth header type is `Bearer`, try to extract the token.
		if bearer, ok := strings.CutPrefix(authHeader[0], "Bearer"); ok {
			token = strings.TrimSpace(bearer)
		}
	}

	// If auth cookie is present try to extract cookie value.
	if len(cookieHeader) != 0 {
		cookies := strings.Split(cookieHeader[0], ";")
		for _, c := range cookies {
			// The name of the cookie is defined in `./build/ansible/roles/grafana/files/grafana.ini`.
			if auth, ok := strings.CutPrefix(strings.TrimSpace(c), "pmm_session="); ok {
				cookie = auth
			}
		}
	}

	params := &dump.Params{
		Token:        token,
		Cookie:       cookie,
		User:         user,
		Password:     password,
		ServiceNames: req.ServiceNames,
		ExportQAN:    req.ExportQan,
		IgnoreLoad:   req.IgnoreLoad,
	}

	if req.StartTime != nil {
		startTime := req.StartTime.AsTime()
		params.StartTime = &startTime
	}

	if req.EndTime != nil {
		endTime := req.EndTime.AsTime()
		params.EndTime = &endTime
	}

	if params.StartTime != nil && params.EndTime != nil {
		if params.StartTime.After(*params.EndTime) {
			return nil, status.Error(codes.InvalidArgument, "Dump start time can't be greater than end time")
		}
	}

	dumpID, err := s.dumpService.StartDump(params)
	if err != nil {
		return nil, err
	}

	return &dumpv1beta1.StartDumpResponse{DumpId: dumpID}, nil
}

// ListDumps lists dumps based on the provided context and request.
func (s *Service) ListDumps(_ context.Context, _ *dumpv1beta1.ListDumpsRequest) (*dumpv1beta1.ListDumpsResponse, error) {
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

// DeleteDump deletes a dump based on the provided context and request.
func (s *Service) DeleteDump(_ context.Context, req *dumpv1beta1.DeleteDumpRequest) (*dumpv1beta1.DeleteDumpResponse, error) {
	for _, id := range req.DumpIds {
		if err := s.dumpService.DeleteDump(id); err != nil {
			return nil, err
		}
	}

	return &dumpv1beta1.DeleteDumpResponse{}, nil
}

// GetDumpLogs retrieves dump logs based on the provided context and request.
func (s *Service) GetDumpLogs(_ context.Context, req *dumpv1beta1.GetDumpLogsRequest) (*dumpv1beta1.GetDumpLogsResponse, error) {
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

	res := &dumpv1beta1.GetDumpLogsResponse{
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

// UploadDump uploads a dump based on the provided context and request.
func (s *Service) UploadDump(_ context.Context, req *dumpv1beta1.UploadDumpRequest) (*dumpv1beta1.UploadDumpResponse, error) {
	filePaths, err := s.dumpService.GetFilePathsForDumps(req.DumpIds)
	if err != nil {
		return nil, err
	}

	if req.SftpParameters == nil {
		return nil, status.Error(codes.InvalidArgument, "SFTP parameters are missing.")
	}

	var config ssh.Config
	config.SetDefaults()
	config.KeyExchanges = append(config.KeyExchanges,
		"diffie-hellman-group-exchange-sha256",
		"diffie-hellman-group-exchange-sha1")
	conf := &ssh.ClientConfig{
		User: req.SftpParameters.User,
		Auth: []ssh.AuthMethod{
			ssh.Password(req.SftpParameters.Password),
		},
		// We can't check host key
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), //nolint:gosec
		Config:          config,
	}

	sshClient, err := ssh.Dial("tcp", req.SftpParameters.Address, conf)
	if err != nil {
		return nil, errors.Wrap(err, "failed to open TCP connection to SFTP server")
	}
	defer sshClient.Close() //nolint:errcheck

	sftpClient, err := sftp.NewClient(sshClient)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create SFTP client")
	}
	defer sftpClient.Close() //nolint:errcheck

	for _, filePath := range filePaths {
		if err = s.uploadFile(sftpClient, filePath, req.SftpParameters.Directory); err != nil {
			return nil, errors.Wrap(err, "failed to upload file on SFTP server")
		}
	}

	return &dumpv1beta1.UploadDumpResponse{}, nil
}

func (s *Service) uploadFile(client *sftp.Client, localFilePath, remoteDir string) error {
	fileName := filepath.Base(localFilePath)
	remoteFilePath := path.Join(remoteDir, fileName)

	nf, err := client.OpenFile(remoteFilePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC)
	if err != nil {
		return errors.Wrap(err, "failed to create file on SFTP server")
	}

	f, err := os.Open(localFilePath) //nolint:gosec
	if err != nil {
		return errors.Wrap(err, "failed to open dump file")
	}
	defer func() {
		if err := f.Close(); err != nil {
			s.l.Errorf("Failed to close file: %+v", err)
		}
	}()
	if _, err = bufio.NewReader(f).WriteTo(nf); err != nil {
		return errors.Wrap(err, "failed to write dump file on SFTP server")
	}

	return nil
}

func convertDump(dump *models.Dump) (*dumpv1beta1.Dump, error) {
	ds, err := convertDumpStatus(dump.Status)
	if err != nil {
		return nil, errors.Wrap(err, "failed to convert dump ds")
	}

	d := &dumpv1beta1.Dump{
		DumpId:       dump.ID,
		Status:       ds,
		ServiceNames: dump.ServiceNames,
		CreatedAt:    timestamppb.New(dump.CreatedAt),
	}

	if dump.StartTime != nil {
		d.StartTime = timestamppb.New(*dump.StartTime)
	}

	if dump.EndTime != nil {
		d.EndTime = timestamppb.New(*dump.EndTime)
	}

	return d, nil
}

func convertDumpStatus(status models.DumpStatus) (dumpv1beta1.DumpStatus, error) {
	switch status {
	case models.DumpStatusSuccess:
		return dumpv1beta1.DumpStatus_DUMP_STATUS_SUCCESS, nil
	case models.DumpStatusError:
		return dumpv1beta1.DumpStatus_DUMP_STATUS_ERROR, nil
	case models.DumpStatusInProgress:
		return dumpv1beta1.DumpStatus_DUMP_STATUS_IN_PROGRESS, nil
	default:
		return dumpv1beta1.DumpStatus_DUMP_STATUS_UNSPECIFIED, errors.Errorf("invalid status '%s'", status)
	}
}
