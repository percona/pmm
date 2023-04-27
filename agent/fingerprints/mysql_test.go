// Copyright 2019 Percona LLC
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

package fingerprints

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type test struct {
	Query                     string
	ExpectedQuery             string
	ExpectedPlaceHoldersCount uint32
}

func TestMySQL(t *testing.T) {
	sqls := []test{
		{
			Query:                     "SELECT `city` . `CountryCode` , `city` . `Name` FROM `world` . `city` WHERE NAME IN (...) LIMIT ? ",
			ExpectedQuery:             "SELECT `city` . `CountryCode` , `city` . `Name` FROM `world` . `city` WHERE NAME IN :1 LIMIT :2 ",
			ExpectedPlaceHoldersCount: 2,
		},
	}

	for _, sql := range sqls {
		query, placeholdersCount, err := GetMySQLFingerprintPlaceholders(sql.Query)
		require.NoError(t, err)
		assert.Equal(t, sql.ExpectedQuery, query)
		assert.Equal(t, sql.ExpectedPlaceHoldersCount, placeholdersCount)
	}
}
