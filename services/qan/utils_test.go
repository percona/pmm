// pmm-managed
// Copyright (C) 2017 Percona LLC
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published
// by the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program. If not, see <http://www.gnu.org/licenses/>.

package qan

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSanitizeDSN(t *testing.T) {
	uris := map[string]string{
		"admin:abc123@127.0.0.1:100":           "admin:***@127.0.0.1:100",
		"localhost:27017/":                     "localhost:27017",
		"localhost:27017?opt=5":                "localhost:27017",
		"localhost":                            "localhost",
		"admin:abc123@localhost:1,localhost:2": "admin:***@localhost:1,localhost:2",
		"root:qwertyUIOP)(*&^%$#@1@localhost":  "root:***@localhost",
		"root:qwerty:UIOP)(*&^%$#@1@localhost": "root:***@localhost",
		"mysql57:secret_password@tcp(mysql57.ckpwzom1xccn.eu-west-1.rds.amazonaws.com:3306)/": "mysql57:***@tcp(mysql57.ckpwzom1xccn.eu-west-1.rds.amazonaws.com:3306)",
	}
	for uri, expected := range uris {
		t.Run(uri, func(t *testing.T) {
			assert.Equal(t, expected, sanitizeDSN(uri))
		})
	}
}
