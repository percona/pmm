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
	"fmt"

	"gopkg.in/reform.v1"
)

// CreateOmBootstrapSecret stores one bootstrap run's generated MongoDB user and keyFile, encrypted.
func CreateOmBootstrapSecret(q *reform.Querier, secret *OmBootstrapSecret) error {
	err := q.Insert(secret)
	if err != nil {
		return fmt.Errorf("failed to insert OM bootstrap secret: %w", err)
	}
	return nil
}

// ClearOmBootstrapAdminPassword blanks a run's stored administrator password.
func ClearOmBootstrapAdminPassword(q *reform.Querier, runID string) error {
	_, err := q.Exec("UPDATE om_bootstrap_secrets SET admin_password = '' WHERE run_id = $1 AND admin_password <> ''", runID)
	if err != nil {
		return fmt.Errorf("failed to clear OM bootstrap admin password: %w", err)
	}
	return nil
}

// FindOmBootstrapSecretByRunID returns one run's stored secret, decrypted, or ErrNotFound.
func FindOmBootstrapSecretByRunID(q *reform.Querier, runID string) (*OmBootstrapSecret, error) {
	if runID == "" {
		return nil, NewInvalidArgumentError("run_id shouldn't be empty")
	}
	secret := &OmBootstrapSecret{RunID: runID}
	err := q.Reload(secret)
	if err != nil {
		if errors.Is(err, reform.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("failed to select OM bootstrap secret: %w", err)
	}
	return secret, nil
}
