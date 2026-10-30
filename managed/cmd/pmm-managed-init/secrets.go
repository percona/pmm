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
	"github.com/sirupsen/logrus"

	"github.com/percona/pmm/managed/utils/secrets"
)

// runSecrets resolves every credential to a file under secrets.Dir.
//
// It runs from the entrypoint before supervisord starts, so every program finds its credentials there.
func runSecrets() error {
	resolved, err := secrets.Prepare(secrets.OptionsFromEnv())
	if err != nil {
		return err
	}
	for _, r := range resolved {
		logrus.Infof("Credential %s: %s.", r.Name, r.Source)
		if r.Padded {
			logrus.Warnf("Credential %s: the value of %s has leading or trailing whitespace, which PMM Server ignores.", r.Name, r.Var)
		}
	}
	return nil
}
