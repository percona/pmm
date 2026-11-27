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
	"errors"

	"gopkg.in/reform.v1"
)

// FindLogParserPresetByID returns a log parser preset by ID.
func FindLogParserPresetByID(q *reform.Querier, id string) (*LogParserPreset, error) {
	if id == "" {
		return nil, errors.New("empty preset ID")
	}

	s, err := q.FindOneFrom(LogParserPresetTable, "id", id)
	if err != nil {
		if errors.Is(err, reform.ErrNoRows) {
			return nil, nil //nolint:nilnil
		}

		return nil, err
	}

	return s.(*LogParserPreset), nil //nolint:forcetypeassert // reform.FindByPrimaryKeyFrom on LogParserPresetTable guarantees this type
}

// FindLogParserPresetByName returns a log parser preset by name.
func FindLogParserPresetByName(q *reform.Querier, name string) (*LogParserPreset, error) {
	if name == "" {
		return nil, errors.New("empty preset name")
	}

	s, err := q.FindOneFrom(LogParserPresetTable, "name", name)
	if err != nil {
		if errors.Is(err, reform.ErrNoRows) {
			return nil, nil //nolint:nilnil
		}

		return nil, err
	}

	return s.(*LogParserPreset), nil //nolint:forcetypeassert // reform.FindOneFrom on LogParserPresetTable guarantees this type
}

// FindAllLogParserPresets returns all log parser presets ordered by name.
func FindAllLogParserPresets(q *reform.Querier) ([]*LogParserPreset, error) {
	structs, err := q.SelectAllFrom(LogParserPresetTable, "ORDER BY name")
	if err != nil {
		return nil, err
	}

	res := make([]*LogParserPreset, len(structs))
	for i, s := range structs {
		res[i] = s.(*LogParserPreset) //nolint:forcetypeassert // reform.SelectAllFrom on LogParserPresetTable guarantees this type
	}

	return res, nil
}
