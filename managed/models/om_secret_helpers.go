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

	"github.com/google/uuid"
	"gopkg.in/reform.v1"

	"github.com/percona/pmm/managed/utils/encryption"
)

// EncryptOmSecretPayload returns a copy with payload encrypted for storage.
func EncryptOmSecretPayload(secret OmSecret) (OmSecret, error) {
	ciphertext, err := encryption.Encrypt(secret.Payload)
	if err != nil {
		return OmSecret{}, fmt.Errorf("payload: %w", err)
	}
	secret.Payload = ciphertext
	return secret, nil
}

// DecryptOmSecretPayload returns a copy with payload decrypted.
func DecryptOmSecretPayload(secret OmSecret) (OmSecret, error) {
	plaintext, err := encryption.Decrypt(secret.Payload)
	if err != nil {
		return OmSecret{}, fmt.Errorf("payload: %w", err)
	}
	secret.Payload = plaintext
	return secret, nil
}

// CreateOmSecret is the only sanctioned writer of om_secrets: it encrypts payload and fails the
// write if encryption fails, so a plaintext payload is never persisted.
func CreateOmSecret(q *reform.Querier, secret *OmSecret) error {
	if secret.ID == "" {
		secret.ID = uuid.New().String()
	}
	encrypted, err := EncryptOmSecretPayload(*secret)
	if err != nil {
		return fmt.Errorf("failed to encrypt OM secret: %w", err)
	}
	err = q.Insert(&encrypted)
	if err != nil {
		return fmt.Errorf("failed to insert OM secret: %w", err)
	}
	decrypted, err := DecryptOmSecretPayload(encrypted)
	if err != nil {
		return fmt.Errorf("failed to decrypt OM secret just inserted: %w", err)
	}
	*secret = decrypted
	return nil
}

// FindOmSecretByID returns one secret, payload decrypted, or ErrNotFound.
func FindOmSecretByID(q *reform.Querier, id string) (*OmSecret, error) {
	if id == "" {
		return nil, NewInvalidArgumentError("id shouldn't be empty")
	}
	secret := &OmSecret{ID: id}
	err := q.Reload(secret)
	if err != nil {
		if errors.Is(err, reform.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("failed to select OM secret: %w", err)
	}
	decrypted, err := DecryptOmSecretPayload(*secret)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt OM secret: %w", err)
	}
	return &decrypted, nil
}
