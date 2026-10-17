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

package secrets

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// RotatedEnvPath is the file with the values that the variables of PMM Server given as Vault references have now,
// for those of them rotated in Vault since PMM Server started. The programs that read them are started through
// "pmm-managed-init exec-env", which sets them, so that a restart of the program applies a rotated secret
// without a restart of PMM Server. It is outside /srv, and the entrypoint removes it at every start: the
// variables have their values from Vault then.
const RotatedEnvPath = "/tmp/pmm-secrets/rotated-env.json"

// rotatedEnvDirPermissions let only PMM Server's user into the directory of RotatedEnvPath.
const rotatedEnvDirPermissions = 0o700

// ReadRotatedEnv returns the rotated values of the variables in the file at path, none if there is no file.
func ReadRotatedEnv(path string) (map[string]string, error) {
	b, err := os.ReadFile(path) //nolint:gosec
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil //nolint:nilnil
	}
	if err != nil {
		return nil, err
	}

	var env map[string]string
	err = json.Unmarshal(b, &env)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	return env, nil
}

// WriteRotatedEnv replaces the file at path with one that only its owner can read, with the rotated values of
// the variables.
func WriteRotatedEnv(path string, env map[string]string) error {
	b, err := json.Marshal(env)
	if err != nil {
		return err
	}

	dir := filepath.Dir(path)
	err = os.MkdirAll(dir, rotatedEnvDirPermissions)
	if err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, "."+filepath.Base(path)+".*")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name()) //nolint:errcheck

	_, err = tmp.Write(b)
	if err == nil {
		err = tmp.Close()
	} else {
		_ = tmp.Close()
	}
	if err != nil {
		return err
	}
	return os.Rename(tmp.Name(), path)
}

// WithRotatedEnv returns environ, a list of name=value, with the variables of env set to their rotated values.
func WithRotatedEnv(environ []string, env map[string]string) []string {
	res := make([]string, 0, len(environ)+len(env))
	for _, kv := range environ {
		name, _, _ := strings.Cut(kv, "=")
		if _, rotated := env[name]; !rotated {
			res = append(res, kv)
		}
	}
	for name, value := range env {
		res = append(res, name+"="+value)
	}
	return res
}
