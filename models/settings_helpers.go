// pmm-managed
// Copyright (C) 2017 Percona LLC
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
	"encoding/json"

	"github.com/pkg/errors"
	"gopkg.in/reform.v1"
)

// GetSettings returns current PMM Server settings.
func GetSettings(q reform.DBTX) (*Settings, error) {
	var b []byte
	if err := q.QueryRow("SELECT settings FROM settings").Scan(&b); err != nil {
		return nil, errors.Wrap(err, "failed to select settings")
	}

	var s Settings
	if err := json.Unmarshal(b, &s); err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal settings")
	}

	s.fillDefaults()
	return &s, nil
}

// SaveSettings saves PMM Server settings.
// It may modify passed settings to fill defaults.
func SaveSettings(q reform.DBTX, s *Settings) error {
	s.fillDefaults()

	b, err := json.Marshal(s)
	if err != nil {
		return errors.Wrap(err, "failed to marshal settings")
	}

	_, err = q.Exec("UPDATE settings SET settings = $1", b)
	if err != nil {
		return errors.Wrap(err, "failed to update settings")
	}

	return nil
}
