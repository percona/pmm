// Copyright (C) 2024 Percona LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//  http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package version

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParsePostgreSQLVersion(t *testing.T) {
	for v, expected := range map[string]string{
		"PostgreSQL 12beta2 (Debian 12~beta2-1.pgdg100+1) on x86_64-pc-linux-gnu, compiled by gcc (Debian 8.3.0-6) 8.3.0, 64-bit":              "12",
		"PostgreSQL 10.9 (Debian 10.9-1.pgdg90+1) on x86_64-pc-linux-gnu, compiled by gcc (Debian 6.3.0-18+deb9u1) 6.3.0 20170516, 64-bit":     "10",
		"PostgreSQL 9.4.23 on x86_64-pc-linux-gnu (Debian 9.4.23-1.pgdg90+1), compiled by gcc (Debian 6.3.0-18+deb9u1) 6.3.0 20170516, 64-bit": "9.4",
	} {
		t.Run(v, func(t *testing.T) {
			actual := ParsePostgreSQLVersion(v)
			assert.Equal(t, expected, actual, "%s", v)
		})
	}
}
