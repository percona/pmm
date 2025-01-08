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

// Package grafana provides util functions related to grafana functionality.
package grafana

import (
	"crypto/md5" //nolint:gosec
	"fmt"
)

// SanitizeSAName is used for sanitize name and it's length for service accounts.
// Max length of service account name is 190 chars (limit in Grafana Postgres DB).
// However, prefix added by grafana is counted too. Prefix is sa-{orgID}-.
// Bare minimum is 5 chars reserved (orgID is <10, like sa-1-) and could be more depends
// on orgID number. Let's reserve 10 chars. It will cover almost one million orgIDs.
// Sanitizing, ensure its length by hashing postfix when length is exceeded.
// MD5 is used because it has fixed length 32 chars.
//
// Be aware that the same method is implemented in the Grafana repo, and all changes should be reflected there as well!
func SanitizeSAName(name string) string {
	if len(name) <= 180 {
		return name
	}

	return fmt.Sprintf("%s%x", name[:148], md5.Sum([]byte(name[148:]))) //nolint:gosec
}
