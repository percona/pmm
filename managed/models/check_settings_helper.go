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

package models

import (
	"github.com/pkg/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gopkg.in/reform.v1"
)

// FindCheckSettings returns all CheckSettings stored in the table.
func FindCheckSettings(q *reform.Querier) (map[string]Interval, error) {
	rows, err := q.SelectAllFrom(CheckSettingsTable, "")
	if err != nil {
		if errors.Is(err, reform.ErrNoRows) {
			return nil, err
		}
		return nil, errors.WithStack(err)
	}

	cs := make(map[string]Interval)
	for _, r := range rows {
		state := r.(*CheckSettings) //nolint:forcetypeassert
		cs[state.Name] = state.Interval
	}
	return cs, nil
}

// FindCheckSettingsByName finds CheckSettings by check name.
func FindCheckSettingsByName(q *reform.Querier, name string) (*CheckSettings, error) {
	if name == "" {
		return nil, status.Error(codes.InvalidArgument, "Empty Check name.")
	}

	cs := &CheckSettings{Name: name}
	err := q.Reload(cs)
	if err != nil {
		if errors.Is(err, reform.ErrNoRows) {
			return nil, err
		}
		return nil, errors.WithStack(err)
	}

	return cs, nil
}

// CreateCheckSettings persists CheckSettings.
func CreateCheckSettings(q *reform.Querier, name string, interval Interval) (*CheckSettings, error) {
	row := &CheckSettings{
		Name:     name,
		Interval: interval,
	}

	if err := q.Insert(row); err != nil {
		return nil, errors.Wrap(err, "failed to create check setting")
	}

	return row, nil
}

// ChangeCheckSettings updates the interval of a check setting if already present.
func ChangeCheckSettings(q *reform.Querier, name string, interval Interval) (*CheckSettings, error) {
	row, err := FindCheckSettingsByName(q, name)
	if err != nil {
		return nil, err
	}

	row.Interval = interval

	if err := q.Update(row); err != nil {
		return nil, errors.Wrap(err, "failed to update check setting")
	}

	return row, nil
}
