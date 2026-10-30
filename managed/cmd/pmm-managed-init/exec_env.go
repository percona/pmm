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
	"fmt"
	"os"
	"os/exec"
	"syscall"

	"github.com/percona/pmm/managed/utils/secrets"
)

// execEnvCommand is the argument that makes pmm-managed-init run the program that follows it with the values of
// the PMM Server secrets rotated in Vault since PMM Server started, see secrets.RotatedEnvPath.
const execEnvCommand = "exec-env"

// runExecEnv replaces pmm-managed-init with the program args name, with the rotated values of the variables in the
// file at rotatedPath set. It only returns if it could not.
func runExecEnv(args []string, rotatedPath string) error {
	if len(args) == 0 {
		return fmt.Errorf("%s needs the program to run", execEnvCommand)
	}

	env := os.Environ()
	rotated, err := secrets.ReadRotatedEnv(rotatedPath)
	if err != nil {
		// Run with the values PMM Server started with rather than not at all.
		fmt.Fprintf(os.Stderr, "Cannot read the PMM Server secrets rotated in Vault, using those it started with: %s.\n", err)
	} else {
		env = secrets.WithRotatedEnv(env, rotated)
	}

	path, err := exec.LookPath(args[0])
	if err != nil {
		return err
	}
	return syscall.Exec(path, args, env) //nolint:gosec
}
