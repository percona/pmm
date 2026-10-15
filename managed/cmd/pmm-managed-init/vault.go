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

package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/percona/pmm/managed/utils/encryption"
	"github.com/percona/pmm/managed/utils/secrets"
	"github.com/percona/pmm/utils/vault"
)

// vaultEnvCommand is the argument that makes pmm-managed-init print the Vault exports instead of its usual work.
const vaultEnvCommand = "vault-env"

const vaultEnvTimeout = time.Minute

var envNameRE = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

// vaultEnvPrefixes are the prefixes of the variables that may be Vault references: those of PMM Server,
// Grafana, and the VictoriaMetrics components, whose credentials an HA deployment sets.
var vaultEnvPrefixes = []string{"PMM_", "GF_", "VM_", "VMAGENT_"} //nolint:gochecknoglobals

// vaultEnv writes to out a shell `export` command for every variable of environ with one of
// vaultEnvPrefixes whose value is a vault:<path>#<key> reference, with the value read from Vault. The
// entrypoint of PMM Server evaluates them before anything reads those variables. The PMM_VAULT_*
// variables configure Vault itself and are never resolved, and neither are the PMM_AGENT_* ones of
// PMM Server's pmm-agent, which reads its references itself and would save the values to its
// configuration file otherwise, nor the encryption key, which pmm-managed reads itself to keep it
// out of the environment of every component.
func vaultEnv(ctx context.Context, cfg vault.Config, environ []string, out io.Writer) error {
	var refs [][2]string
	for _, kv := range environ {
		name, value, _ := strings.Cut(kv, "=")
		hasPrefix := slices.ContainsFunc(vaultEnvPrefixes, func(prefix string) bool { return strings.HasPrefix(name, prefix) })
		skipped := strings.HasPrefix(name, "PMM_VAULT_") || strings.HasPrefix(name, "PMM_AGENT_") || name == encryption.KeyEnvVar
		if !vault.IsRef(value) || !hasPrefix || skipped {
			continue
		}
		if !envNameRE.MatchString(name) {
			return fmt.Errorf("invalid environment variable name '%s'", name)
		}
		refs = append(refs, [2]string{name, value})
	}

	if len(refs) == 0 {
		return nil
	}
	if !cfg.Enabled() {
		return fmt.Errorf("%s is a Vault reference, but PMM_VAULT_ADDRESS is not set", refs[0][0])
	}

	client, err := vault.New(cfg)
	if err != nil {
		return err
	}

	for _, ref := range refs {
		value, err := client.GetRef(ctx, ref[1])
		if err != nil {
			return fmt.Errorf("failed to read %s from Vault: %w", ref[0], err)
		}

		_, err = fmt.Fprintf(out, "export %s=%s\n", ref[0], shellQuote(value))
		if err != nil {
			return err
		}
	}
	return nil
}

// shellQuote quotes s for a POSIX shell: inside single quotes only the single quote itself needs escaping.
func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

// vaultPutCommand is the argument that makes pmm-managed-init store a secret given on stdin in Vault,
// under the vault:<path>#<key> reference that follows it.
const vaultPutCommand = "vault-put"

// vaultPut stores the value read from in under the vault:<path>#<key> reference ref. A single trailing
// newline is not part of the value.
func vaultPut(ctx context.Context, cfg vault.Config, ref string, in io.Reader) error {
	if !cfg.Enabled() {
		return errors.New("PMM_VAULT_ADDRESS is not set")
	}

	b, err := io.ReadAll(in)
	if err != nil {
		return err
	}

	client, err := vault.New(cfg)
	if err != nil {
		return err
	}
	return client.WriteRef(ctx, ref, strings.TrimSuffix(string(b), "\n"))
}

func runVaultPut(ref string) error {
	ctx, cancel := context.WithTimeout(context.Background(), vaultEnvTimeout)
	defer cancel()

	cfg, err := secrets.ConfigFromEnv(os.Getenv)
	if err != nil {
		return err
	}
	return vaultPut(ctx, cfg, ref, os.Stdin)
}

func runVaultEnv() error {
	ctx, cancel := context.WithTimeout(context.Background(), vaultEnvTimeout)
	defer cancel()

	cfg, err := secrets.ConfigFromEnv(os.Getenv)
	if err != nil {
		return err
	}
	return vaultEnv(ctx, cfg, os.Environ(), os.Stdout)
}
