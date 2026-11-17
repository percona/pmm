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

	"github.com/sirupsen/logrus"

	"github.com/percona/pmm/managed/utils/secrets"
)

// runSecrets resolves every credential to a file under secrets.Dir before supervisord starts the programs.
func runSecrets() error {
	resolved, err := secrets.Prepare(secrets.OptionsFromEnv())
	if err != nil {
		return err
	}
	for _, r := range resolved {
		logrus.Infof("Credential %s: %s.", r.Name, r.Source)
		switch {
		case r.Kept:
			logrus.Warnf("Credential %s: %s has leading or trailing whitespace, so it stays in the environment, "+
				"where Grafana reads it as written. Don't move this key to a file, because a file can't keep the whitespace.", r.Name, r.Var)
		case r.Padded:
			logrus.Warnf("Credential %s: %s has leading or trailing whitespace, which PMM Server removed from the password. "+
				"Remove the whitespace from the variable.", r.Name, r.Var)
		}
		if r.Ignored != "" {
			logrus.Warnf("Credential %s: %s and %s have different values. PMM Server uses %s. "+
				"Remove %s, or set it to the same value.", r.Name, r.Ignored, r.Var, r.Var, r.Ignored)
		}
		if r.Dropped != "" {
			logrus.Warnf("Credential %s: %s has a password but no username. VictoriaMetrics uses a password only together "+
				"with a username, so PMM Server drops it. Remove the password from %s, or add the username.", r.Name, r.Dropped, r.Dropped)
		}
	}
	return nil
}

// runVMURL prints PMM_VM_URL for PMM's programs, for the entrypoint to export.
func runVMURL() error {
	_, err := fmt.Fprintln(os.Stdout, vmURLForPrograms(os.Getenv(secrets.VMURLEnvVar)))
	return err
}

// vmURLForPrograms returns raw without its password: the secrets step stored the password in its file, or dropped it
// for having no username. A URL that doesn't parse holds no password that the secrets step found, so it passes
// unchanged, and pmm-managed reports it.
func vmURLForPrograms(raw string) string {
	u, err := secrets.URLWithoutPassword(raw)
	if err != nil {
		return raw
	}
	return u
}
