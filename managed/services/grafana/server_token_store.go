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

package grafana

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"
	"gopkg.in/reform.v1"

	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/managed/utils/encryption"
)

// ServerTokenStore loads and persists the Grafana service-account token used for
// server-initiated API calls (annotations, alerting provisioning).
type ServerTokenStore interface {
	// Load returns the stored token, or "" if none is stored yet.
	Load(ctx context.Context) (string, error)
	// Save persists the token.
	Save(ctx context.Context, token string) error
}

// dbServerTokenStore persists the token, encrypted, in PMM settings.
type dbServerTokenStore struct {
	db *reform.DB
}

// NewServerTokenStore returns a ServerTokenStore backed by the PMM database; the token is
// encrypted at rest via the PMM encryption key.
func NewServerTokenStore(db *reform.DB) ServerTokenStore {
	return &dbServerTokenStore{db: db}
}

func (s *dbServerTokenStore) Load(_ context.Context) (string, error) {
	settings, err := models.GetSettings(s.db.Querier)
	if err != nil {
		return "", err
	}
	if settings.PMMServiceToken == "" {
		return "", nil
	}
	token, err := encryption.Decrypt(settings.PMMServiceToken)
	if err != nil {
		// Decrypt failure (e.g. after an encryption-key rotation) is non-fatal: report no token
		// so the caller mints a fresh one and persists it with the current key.
		logrus.WithField("component", "grafana/server-token").
			Warnf("Failed to decrypt the Grafana service token, minting a new one: %s", err)
		return "", nil
	}
	return token, nil
}

func (s *dbServerTokenStore) Save(ctx context.Context, token string) error {
	encrypted, err := encryption.Encrypt(token)
	if err != nil {
		return fmt.Errorf("failed to encrypt Grafana service token: %w", err)
	}
	return s.db.InTransactionContext(ctx, nil, func(tx *reform.TX) error {
		settings, err := models.GetSettings(tx.Querier)
		if err != nil {
			return err
		}
		settings.PMMServiceToken = encrypted
		return models.SaveSettings(tx.Querier, settings)
	})
}
