// pmm-managed
// Copyright (C) 2017 Percona LLC
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

package qan

import (
	"fmt"
	"strings"
)

// sanitizeDSN remove password from DSN.
func sanitizeDSN(dsn string) string {
	dsn = strings.TrimRight(strings.Split(dsn, "?")[0], "/")
	if strings.Index(dsn, "@") > 0 {
		dsnParts := strings.Split(dsn, "@")
		userPart := dsnParts[0]
		hostPart := ""
		if len(dsnParts) > 1 {
			hostPart = dsnParts[len(dsnParts)-1]
		}
		userPasswordParts := strings.Split(userPart, ":")
		dsn = fmt.Sprintf("%s:***@%s", userPasswordParts[0], hostPart)
	}
	return dsn
}
