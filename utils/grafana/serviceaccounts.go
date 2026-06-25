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
func SanitizeSAName(name string) string {
	if len(name) <= 180 {
		return name
	}

	return fmt.Sprintf("%s%x", name[:148], md5.Sum([]byte(name[148:]))) //nolint:gosec
}
