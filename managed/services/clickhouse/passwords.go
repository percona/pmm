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

package clickhouse

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/sirupsen/logrus"
)

// passwordsDropIn is the users.d file that sets the passwords of the built-in users, see ApplyPasswords.
const passwordsDropIn = "users.d/pmm-passwords.xml" //nolint:gosec

// builtinUsers are the users of the built-in ClickHouse that PMM connects as, by the variable that sets
// the name of each, its default name, and the variable that sets its password.
var builtinUsers = []struct{ nameEnv, defaultName, passwordEnv string }{ //nolint:gochecknoglobals
	{"PMM_CLICKHOUSE_USER", "default", "PMM_CLICKHOUSE_PASSWORD"},
	{"PMM_CLICKHOUSE_DATASOURCE_USER", "grafana", "PMM_CLICKHOUSE_DATASOURCE_PASSWORD"},
}

// ApplyPasswords makes the built-in ClickHouse accept the passwords that PMM_CLICKHOUSE_PASSWORD and
// PMM_CLICKHOUSE_DATASOURCE_PASSWORD give its clients, which may have been read from Vault: it writes
// their SHA-256 hashes to a users.d file, or removes the file when neither is set and the passwords
// of users.xml apply. The passwords themselves are never written.
func ApplyPasswords(getenv func(string) string) error {
	return applyPasswordsAt(clickHouseConfigDir, getenv)
}

func applyPasswordsAt(dir string, getenv func(string) string) error {
	path := filepath.Join(dir, passwordsDropIn)

	var users strings.Builder
	for _, u := range builtinUsers {
		password := getenv(u.passwordEnv)
		if password == "" {
			continue
		}
		if name := getenv(u.nameEnv); name != "" && name != u.defaultName {
			// Not a user of users.xml, whose settings a password alone would not make.
			logrus.Warnf("%s is set for the user %s, which the built-in ClickHouse does not have.", u.passwordEnv, name)
			continue
		}

		sum := sha256.Sum256([]byte(password))
		fmt.Fprintf(&users, "        <%s>\n            <password_sha256_hex replace=\"replace\">%s</password_sha256_hex>\n        </%s>\n",
			u.defaultName, hex.EncodeToString(sum[:]), u.defaultName)
	}

	if users.Len() == 0 {
		err := os.Remove(path)
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("cannot remove %s: %w", path, err)
		}
		return nil
	}

	content := "<!-- Written by pmm-managed-init from PMM_CLICKHOUSE_PASSWORD and PMM_CLICKHOUSE_DATASOURCE_PASSWORD. -->\n" +
		"<clickhouse>\n    <users>\n" + users.String() + "    </users>\n</clickhouse>\n"
	err := os.WriteFile(path, []byte(content), 0o600) //nolint:mnd
	if err != nil {
		return fmt.Errorf("cannot write %s: %w", path, err)
	}
	return nil
}
