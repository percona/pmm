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
	"github.com/pkg/errors"
	"gopkg.in/reform.v1"
)

// GetPerconaSSODetails returns PerconaSSODetails if there are any, error otherwise.
func GetPerconaSSODetails(q *reform.Querier) (*PerconaSSODetails, error) {
	ssoDetails, err := q.SelectOneFrom(PerconaSSODetailsView, "")
	if err != nil {
		return nil, errors.Wrap(err, "failed to get Percona SSO Details")
	}
	return ssoDetails.(*PerconaSSODetails), nil
}

// DeletePerconaSSODetails removes all stored DeletePerconaSSODetails.
func DeletePerconaSSODetails(q *reform.Querier) error {
	_, err := q.DeleteFrom(PerconaSSODetailsView, "")
	if err != nil {
		return errors.Wrap(err, "failed to delete Percona SSO Details")
	}
	return nil
}

// InsertPerconaSSODetails inserts a new Percona SSO details.
func InsertPerconaSSODetails(q *reform.Querier, ssoDetails *PerconaSSODetails) error {
	if err := q.Insert(ssoDetails); err != nil {
		return errors.Wrap(err, "failed to insert Percona SSO Details")
	}
	return nil
}
