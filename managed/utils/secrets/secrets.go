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

// Package secrets reads the secrets that PMM Server uses itself from Vault, for the settings that hold a
// vault:<path>#<key> reference instead of the secret. The credentials of databases are not among them:
// pmm-agent reads those with its own Vault identity, and they never reach PMM Server.
package secrets

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/percona/pmm/utils/vault"
)

// ErrNotConfigured is returned for a reference when PMM Server has no Vault configured.
var ErrNotConfigured = errors.New("PMM Server has no Vault configured (see PMM_VAULT_ADDRESS)")

// Store reads the secrets, see vault.Client.
type Store interface {
	Get(ctx context.Context, path, key string) (string, error)
}

type holder struct {
	store    Store
	cacheTTL time.Duration
}

var current atomic.Pointer[holder] //nolint:gochecknoglobals

// Configure sets the Vault PMM Server reads its secrets from, and how long it reuses them; nil turns it off.
func Configure(s Store, cacheTTL time.Duration) {
	if s == nil {
		current.Store(nil)
		return
	}
	current.Store(&holder{store: s, cacheTTL: cacheTTL})
}

// Configured reports whether PMM Server reads secrets from Vault.
func Configured() bool {
	return current.Load() != nil
}

// CacheTTL returns how long PMM Server reuses the secrets it read, and so how often it checks them for changes.
func CacheTTL() time.Duration {
	h := current.Load()
	if h == nil || h.cacheTTL <= 0 {
		return vault.DefaultCacheTTL
	}
	return h.cacheTTL
}

// Resolve returns s, or the secret it refers to if it is a vault:<path>#<key> reference.
func Resolve(ctx context.Context, s string) (string, error) {
	if !vault.IsRef(s) {
		return s, nil
	}

	ref, err := vault.ParseRef(s)
	if err != nil {
		return "", err
	}

	h := current.Load()
	if h == nil {
		return "", fmt.Errorf("cannot read %s: %w", s, ErrNotConfigured)
	}

	v, err := h.store.Get(ctx, ref.Path, ref.Key)
	if err != nil {
		return "", fmt.Errorf("cannot read %s from Vault: %w", s, err)
	}
	return v, nil
}

// ResolveAll replaces each of the given strings that holds a reference with the secret it refers to.
func ResolveAll(ctx context.Context, strs ...*string) error {
	for _, s := range strs {
		v, err := Resolve(ctx, *s)
		if err != nil {
			return err
		}
		*s = v
	}
	return nil
}

// ConfigFromEnv returns the Vault configuration of PMM Server, given by the PMM_VAULT_* variables.
func ConfigFromEnv(getenv func(string) string) (vault.Config, error) {
	skipVerify, _ := strconv.ParseBool(getenv("PMM_VAULT_TLS_SKIP_VERIFY"))

	cfg := vault.Config{
		Address:       getenv("PMM_VAULT_ADDRESS"),
		Namespace:     getenv("PMM_VAULT_NAMESPACE"),
		CACert:        getenv("PMM_VAULT_CA_CERT"),
		TLSSkipVerify: skipVerify,
		AuthMethod:    getenv("PMM_VAULT_AUTH_METHOD"),
		AuthMount:     getenv("PMM_VAULT_AUTH_MOUNT"),
		Token:         getenv("PMM_VAULT_TOKEN"),
		TokenFile:     getenv("PMM_VAULT_TOKEN_FILE"),
		RoleID:        getenv("PMM_VAULT_ROLE_ID"),
		SecretID:      getenv("PMM_VAULT_SECRET_ID"),
		SecretIDFile:  getenv("PMM_VAULT_SECRET_ID_FILE"),
		Role:          getenv("PMM_VAULT_ROLE"),
		JWTFile:       getenv("PMM_VAULT_JWT_FILE"),
		AWSRegion:     getenv("PMM_VAULT_AWS_REGION"),
		AWSServerID:   getenv("PMM_VAULT_AWS_SERVER_ID"),
	}

	if ttl := getenv("PMM_VAULT_CACHE_TTL"); ttl != "" {
		d, err := time.ParseDuration(ttl)
		if err != nil {
			return vault.Config{}, fmt.Errorf("invalid PMM_VAULT_CACHE_TTL: %w", err)
		}
		cfg.CacheTTL = d
	}
	return cfg, nil
}
