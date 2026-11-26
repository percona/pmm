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

package om

import (
	"context"
	"encoding/json"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/structpb"

	omv1 "github.com/percona/pmm/api/om/v1"
	"github.com/percona/pmm/managed/models"
)

// settingsPrincipal is the recorded actor for a settings change until caller identity is wired in.
const settingsPrincipal = "operator"

// GetSettings resolves the Operations policy settings for a scope.
func (s *Service) GetSettings(_ context.Context, req *omv1.GetSettingsRequest) (*omv1.GetSettingsResponse, error) {
	settings, err := models.GetOmSettings(s.db.Querier, s.settingsRegistry, req.ClusterId)
	if err != nil {
		return nil, settingsError(err, codes.Internal)
	}
	return &omv1.GetSettingsResponse{Settings: omSettingsToProto(settings)}, nil
}

// UpdateSettings validates and applies a batch of settings changes for a scope, then returns the
// settings as they now stand.
func (s *Service) UpdateSettings(ctx context.Context, req *omv1.UpdateSettingsRequest) (*omv1.UpdateSettingsResponse, error) {
	values, err := structToRawMap(req.GetValues())
	if err != nil {
		return nil, settingsError(err, codes.InvalidArgument)
	}
	err = models.UpdateOmSettings(ctx, s.db, s.settingsRegistry, req.ClusterId, nil, settingsPrincipal, values)
	if err != nil {
		return nil, settingsError(err, codes.InvalidArgument)
	}
	current, err := models.GetOmSettings(s.db.Querier, s.settingsRegistry, req.ClusterId)
	if err != nil {
		return nil, settingsError(err, codes.Internal)
	}
	return &omv1.UpdateSettingsResponse{Settings: omSettingsToProto(current)}, nil
}

// DeleteSettingOverride removes a scope's override for one setting.
func (s *Service) DeleteSettingOverride(ctx context.Context, req *omv1.DeleteSettingOverrideRequest) (*omv1.DeleteSettingOverrideResponse, error) {
	removed, err := models.DeleteOmSettingOverride(ctx, s.db, s.settingsRegistry, req.ClusterId, settingsPrincipal, nil, req.GetKey())
	if err != nil {
		return nil, settingsError(err, codes.InvalidArgument)
	}
	return &omv1.DeleteSettingOverrideResponse{Removed: removed}, nil
}

// settingsError preserves an existing gRPC status or wraps a plain error with the given code.
func settingsError(err error, code codes.Code) error {
	_, ok := status.FromError(err)
	if ok {
		return err
	}
	return status.Error(code, err.Error())
}

// structToRawMap converts a protobuf Struct of changes into per-key JSON.
func structToRawMap(s *structpb.Struct) (map[string]json.RawMessage, error) {
	fields := s.GetFields()
	out := make(map[string]json.RawMessage, len(fields))
	for key := range fields {
		raw, err := json.Marshal(fields[key].AsInterface())
		if err != nil {
			return nil, err
		}
		out[key] = raw
	}
	return out, nil
}

// omSettingsToProto maps resolved settings to their proto form; a secret key carries no value.
func omSettingsToProto(settings []models.OmEffectiveSetting) []*omv1.OmSetting {
	out := make([]*omv1.OmSetting, 0, len(settings))
	for i := range settings {
		eff := settings[i]
		p := &omv1.OmSetting{
			Key:         eff.Key,
			Source:      sourceToProto(eff.Source),
			HasOverride: eff.HasOverride,
			Secret:      eff.Secret,
			IsSet:       eff.IsSet,
		}
		if eff.Description != "" {
			p.Description = &eff.Description
		}
		if eff.Value != nil {
			value, err := rawToValue(eff.Value)
			if err == nil {
				p.Value = value
			}
		}
		out = append(out, p)
	}
	return out
}

// rawToValue converts stored JSON into a protobuf Value.
func rawToValue(raw json.RawMessage) (*structpb.Value, error) {
	var decoded any
	err := json.Unmarshal(raw, &decoded)
	if err != nil {
		return nil, err
	}
	return structpb.NewValue(decoded)
}

func sourceToProto(source models.OmSettingSource) omv1.OmSettingSource {
	switch source {
	case models.OmSettingSourceCluster:
		return omv1.OmSettingSource_OM_SETTING_SOURCE_CLUSTER
	case models.OmSettingSourceGlobal:
		return omv1.OmSettingSource_OM_SETTING_SOURCE_GLOBAL
	case models.OmSettingSourceDefault:
		return omv1.OmSettingSource_OM_SETTING_SOURCE_DEFAULT
	default:
		return omv1.OmSettingSource_OM_SETTING_SOURCE_UNSPECIFIED
	}
}
