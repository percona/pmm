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

	"github.com/percona/pmm/managed/utils/encryption"
)

// EncryptOmBootstrapSecret encrypts secret's fields for storage.
func EncryptOmBootstrapSecret(secret OmBootstrapSecret) (OmBootstrapSecret, error) {
	return omBootstrapSecretEncryption(secret, encryption.Encrypt)
}

// DecryptOmBootstrapSecret decrypts secret's fields read from storage.
func DecryptOmBootstrapSecret(secret OmBootstrapSecret) (OmBootstrapSecret, error) {
	return omBootstrapSecretEncryption(secret, encryption.Decrypt)
}

// omBootstrapSecretEncryption stops on the first failure rather than logging and
// keeping the untransformed field: a MongoDB password or keyFile this returns
// still needs handler applied to every field, or CreateOmBootstrapSecret's caller
// would insert a run's plaintext credential next to its correctly-encrypted
// siblings -- exactly the partial-encryption outcome utils/encryption.EncryptItems
// (agents.username/password's own bulk path) never allows either.
func omBootstrapSecretEncryption(secret OmBootstrapSecret, handler func(string) (string, error)) (OmBootstrapSecret, error) {
	username, err := handler(secret.MongoDBUsername)
	if err != nil {
		return OmBootstrapSecret{}, fmt.Errorf("mongodb_username: %w", err)
	}
	secret.MongoDBUsername = username

	password, err := handler(secret.MongoDBPassword)
	if err != nil {
		return OmBootstrapSecret{}, fmt.Errorf("mongodb_password: %w", err)
	}
	secret.MongoDBPassword = password

	keyFile, err := handler(secret.KeyFile)
	if err != nil {
		return OmBootstrapSecret{}, fmt.Errorf("key_file: %w", err)
	}
	secret.KeyFile = keyFile

	return secret, nil
}

// CreateOmBootstrapSecret stores one bootstrap run's generated MongoDB user and
// keyFile, encrypted.
//
// Callers create this exactly once per run, the first time the stepper needs a
// value it did not already have on hand (see OmBootstrapSecret's own doc comment) --
// there is nothing to update afterward, since neither field ever changes for the
// life of a run.
func CreateOmBootstrapSecret(q *reform.Querier, secret *OmBootstrapSecret) error {
	encrypted, err := EncryptOmBootstrapSecret(*secret)
	if err != nil {
		return fmt.Errorf("failed to encrypt OM bootstrap secret: %w", err)
	}
	err = q.Insert(&encrypted)
	if err != nil {
		return fmt.Errorf("failed to insert OM bootstrap secret: %w", err)
	}
	decrypted, err := DecryptOmBootstrapSecret(encrypted)
	if err != nil {
		return fmt.Errorf("failed to decrypt OM bootstrap secret just inserted: %w", err)
	}
	*secret = decrypted
	return nil
}

// FindOmBootstrapSecretByRunID returns one run's stored secret, decrypted, or
// ErrNotFound when the stepper has not generated one for this run yet.
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
	decrypted, err := DecryptOmBootstrapSecret(*secret)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt OM bootstrap secret: %w", err)
	}
	return &decrypted, nil
}
