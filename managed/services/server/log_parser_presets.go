// Copyright (C) 2023 Percona LLC
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program. If not, see <http://www.gnu.org/licenses/>.

package server

import (
	"context"
	"errors"
	"strings"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gopkg.in/reform.v1"

	serverv1 "github.com/percona/pmm/api/server/v1"
	"github.com/percona/pmm/managed/models"
)

func convertLogParserPreset(p *models.LogParserPreset) *serverv1.LogParserPreset {
	if p == nil {
		return nil
	}
	desc := ""
	if p.Description != nil {
		desc = *p.Description
	}
	return &serverv1.LogParserPreset{
		Id:           p.ID,
		Name:         p.Name,
		Description:  desc,
		OperatorYaml: p.OperatorYAML,
		BuiltIn:      p.BuiltIn,
		CreatedAt:    timestamppb.New(p.CreatedAt),
		UpdatedAt:    timestamppb.New(p.UpdatedAt),
	}
}

func mapLogParserPresetErr(err error) error {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, reform.ErrNoRows):
		return status.Error(codes.NotFound, "Log parser preset not found.")
	default:
	}
	msg := err.Error()
	switch {
	case strings.Contains(msg, "already exists"):
		return status.Errorf(codes.AlreadyExists, "%s", msg)
	case strings.Contains(msg, "preset name"), strings.Contains(msg, "operator_yaml"),
		strings.Contains(msg, "operator "), strings.Contains(msg, "empty preset"):
		return status.Errorf(codes.InvalidArgument, "%s", msg)
	case strings.Contains(msg, "cannot delete built-in"):
		return status.Errorf(codes.FailedPrecondition, "%s", msg)
	default:
		return err
	}
}

// ListLogParserPresets returns all log parser presets.
func (s *Server) ListLogParserPresets(_ context.Context, _ *serverv1.ListLogParserPresetsRequest) (*serverv1.ListLogParserPresetsResponse, error) {
	rows, err := models.FindAllLogParserPresets(s.db.Querier)
	if err != nil {
		return nil, err
	}
	out := make([]*serverv1.LogParserPreset, 0, len(rows))
	for _, r := range rows {
		preset := convertLogParserPreset(r)
		ids, uerr := models.ListOtelCollectorAgentIDsReferencingLogParserPreset(s.db.Querier, r.Name)
		if uerr != nil {
			return nil, uerr
		}
		preset.UsageCount = int32(len(ids)) //nolint:gosec
		out = append(out, preset)
	}
	return &serverv1.ListLogParserPresetsResponse{Presets: out}, nil
}

// GetLogParserPreset returns one preset by id.
func (s *Server) GetLogParserPreset(_ context.Context, req *serverv1.GetLogParserPresetRequest) (*serverv1.GetLogParserPresetResponse, error) {
	if strings.TrimSpace(req.GetId()) == "" {
		return nil, status.Error(codes.InvalidArgument, "id is required.")
	}
	row, err := models.FindLogParserPresetByID(s.db.Querier, req.GetId())
	if err != nil {
		return nil, err
	}
	if row == nil {
		return nil, status.Error(codes.NotFound, "Log parser preset not found.")
	}
	return &serverv1.GetLogParserPresetResponse{Preset: convertLogParserPreset(row)}, nil
}

// AddLogParserPreset creates a custom preset.
func (s *Server) AddLogParserPreset(ctx context.Context, req *serverv1.AddLogParserPresetRequest) (*serverv1.AddLogParserPresetResponse, error) {
	row, err := models.CreateLogParserPreset(s.db.Querier, req.GetName(), req.GetDescription(), req.GetOperatorYaml())
	if err != nil {
		return nil, mapLogParserPresetErr(err)
	}
	err = s.agentsState.UpdateAgentsState(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Failed to refresh agents state: %s.", err.Error())
	}
	return &serverv1.AddLogParserPresetResponse{Preset: convertLogParserPreset(row)}, nil
}

// ChangeLogParserPreset updates description and/or operator YAML.
func (s *Server) ChangeLogParserPreset(ctx context.Context, req *serverv1.ChangeLogParserPresetRequest) (*serverv1.ChangeLogParserPresetResponse, error) {
	if strings.TrimSpace(req.GetId()) == "" {
		return nil, status.Error(codes.InvalidArgument, "id is required.")
	}
	if req.Description == nil && req.OperatorYaml == nil {
		return nil, status.Error(codes.InvalidArgument, "At least one of description or operator_yaml must be set.")
	}
	var descPtr *string
	if req.Description != nil {
		descPtr = req.Description
	}
	var yamlPtr *string
	if req.OperatorYaml != nil {
		y := strings.TrimSpace(*req.OperatorYaml)
		yamlPtr = &y
	}
	row, err := models.UpdateLogParserPreset(s.db.Querier, req.GetId(), descPtr, yamlPtr)
	if err != nil {
		if errors.Is(err, reform.ErrNoRows) {
			return nil, status.Error(codes.NotFound, "Log parser preset not found.")
		}
		return nil, mapLogParserPresetErr(err)
	}
	err = s.agentsState.UpdateAgentsState(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Failed to refresh agents state: %s.", err.Error())
	}
	return &serverv1.ChangeLogParserPresetResponse{Preset: convertLogParserPreset(row)}, nil
}

// RemoveLogParserPreset deletes a custom preset that is not in use.
func (s *Server) RemoveLogParserPreset(ctx context.Context, req *serverv1.RemoveLogParserPresetRequest) (*serverv1.RemoveLogParserPresetResponse, error) {
	if strings.TrimSpace(req.GetId()) == "" {
		return nil, status.Error(codes.InvalidArgument, "id is required.")
	}
	row, err := models.FindLogParserPresetByID(s.db.Querier, req.GetId())
	if err != nil {
		return nil, err
	}
	if row == nil {
		return nil, status.Error(codes.NotFound, "Log parser preset not found.")
	}
	if row.BuiltIn {
		return nil, status.Error(codes.FailedPrecondition, "Cannot delete built-in log parser preset.")
	}
	ids, err := models.ListOtelCollectorAgentIDsReferencingLogParserPreset(s.db.Querier, row.Name)
	if err != nil {
		return nil, err
	}
	if len(ids) > 0 {
		return nil, status.Errorf(
			codes.FailedPrecondition,
			"Preset %q is used by OTEL collector agent(s): %s. Remove log_sources first.",
			row.Name,
			strings.Join(ids, ", "),
		)
	}
	err = models.DeleteLogParserPreset(s.db.Querier, req.GetId())
	if err != nil {
		return nil, mapLogParserPresetErr(err)
	}
	err = s.agentsState.UpdateAgentsState(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Failed to refresh agents state: %s.", err.Error())
	}
	return &serverv1.RemoveLogParserPresetResponse{}, nil
}
