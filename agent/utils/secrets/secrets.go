// Copyright (C) 2023 Percona LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package secrets resolves the Vault tokens pmm-managed sends in place of database credentials
// stored in Vault, and the vault:<path>#<key> references in pmm-agent's own configuration.
//
// Tokens arrive in DSNs, exporter arguments, environment and configuration files alike, and they are
// resolved where those are finally used. The Vault client is therefore process-wide, like the TLS
// configurations the MySQL driver keeps: it is set once per configuration load by `pmm-agent run`.
package secrets

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"sync/atomic"

	"github.com/percona/pmm/utils/vault"
)

// Getter returns the value of key in the Vault secret at path.
type Getter interface {
	Get(ctx context.Context, path, key string) (string, error)
}

type holder struct {
	getter Getter
}

var current atomic.Pointer[holder] //nolint:gochecknoglobals

var (
	// ErrUnavailable wraps every error of reading a credential from Vault.
	ErrUnavailable = errors.New("failed to read a credential from Vault")

	// ErrNotConfigured is returned when a credential is stored in Vault, but pmm-agent has no Vault to read it from.
	ErrNotConfigured = errors.New("no Vault is configured for this pmm-agent (see --vault-address and the other --vault-* flags)")
)

// Configure sets the Vault client secrets are read with; nil means Vault is not configured.
func Configure(g Getter) {
	if g == nil {
		current.Store(nil)
		return
	}
	current.Store(&holder{getter: g})
}

func loadHolder() (*holder, error) {
	h := current.Load()
	if h == nil {
		return nil, fmt.Errorf("%w: %w", ErrUnavailable, ErrNotConfigured)
	}
	return h, nil
}

// Resolve replaces the Vault tokens in s with the values they stand for. Alongside the result it returns
// the values it inserted, so that callers can redact them from logs.
func Resolve(ctx context.Context, s string) (string, []string, error) {
	if !vault.HasTokens(s) {
		return s, nil, nil
	}

	h, err := loadHolder()
	if err != nil {
		return "", nil, err
	}

	res, values, err := vault.ReplaceTokens(s, func(path, key string) (string, error) {
		return h.getter.Get(ctx, path, key)
	})
	if err != nil {
		return "", nil, fmt.Errorf("%w: %w", ErrUnavailable, err)
	}
	return res, values, nil
}

// ResolveString is Resolve for callers that do not log the result.
func ResolveString(ctx context.Context, s string) (string, error) {
	res, _, err := Resolve(ctx, s)
	return res, err
}

// ResolveRef returns the value s points to if it is a vault:<path>#<key> reference, and s itself otherwise.
func ResolveRef(ctx context.Context, s string) (string, error) {
	if !vault.IsRef(s) {
		return s, nil
	}

	ref, err := vault.ParseRef(s)
	if err != nil {
		return "", err
	}

	h, err := loadHolder()
	if err != nil {
		return "", err
	}

	v, err := h.getter.Get(ctx, ref.Path, ref.Key)
	if err != nil {
		return "", fmt.Errorf("%w: %w", ErrUnavailable, err)
	}
	return v, nil
}

// Fingerprint returns a digest of strs with their Vault tokens resolved, or an empty string if they
// have none. It changes whenever one of the secrets does.
func Fingerprint(ctx context.Context, strs ...string) (string, error) {
	h := sha256.New()
	var found bool
	for _, s := range strs {
		if !vault.HasTokens(s) {
			continue
		}
		found = true

		res, _, err := Resolve(ctx, s)
		if err != nil {
			return "", err
		}
		h.Write([]byte(res))
		h.Write([]byte{0})
	}

	if !found {
		return "", nil
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}
