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
	DigestText                string
	ExpectedFingerprint       string
	ExpectedPlaceHoldersCount uint32
}

func TestMySQL(t *testing.T) {
	sqls := []test{
		{
			Query:                     "SELECT `city` . `CountryCode` , `city` . `Name` FROM `world` . `city` WHERE NAME IN ('? ? ??? (...)', \"(?+)\") LIMIT ?",
			DigestText:                "SELECT `city` . `CountryCode` , `city` . `Name` FROM `world` . `city` WHERE NAME IN (...) LIMIT ?",
			ExpectedFingerprint:       "SELECT `city` . `CountryCode` , `city` . `Name` FROM `world` . `city` WHERE NAME IN (:1, :2) LIMIT :3",
			ExpectedPlaceHoldersCount: 3,
		},
		{
			Query:                     "SELECT SCHEMA_NAME FROM information_schema.schemata WHERE SCHEMA_NAME NOT IN ('mysql', 'performance_schema', 'information_schema')",
			DigestText:                "SELECT SCHEMA_NAME FROM `information_schema` . `schemata` WHERE SCHEMA_NAME NOT IN (...)",
			ExpectedFingerprint:       "SELECT SCHEMA_NAME FROM `information_schema` . `schemata` WHERE SCHEMA_NAME NOT IN (:1, :2, :3)",
			ExpectedPlaceHoldersCount: 3,
		},
		{
			Query:                     "SELECT productVendor, COUNT(*) FROM products GROUP BY productVendor HAVING COUNT(*) >= 9 ORDER BY COUNT(*) DESC;",
			DigestText:                "SELECT `productVendor` , COUNT ( * ) FROM `products` GROUP BY `productVendor` HAVING COUNT ( * ) >= ? ORDER BY COUNT ( * ) DESC ;",
			ExpectedFingerprint:       "SELECT `productVendor` , COUNT ( * ) FROM `products` GROUP BY `productVendor` HAVING COUNT ( * ) >= :1 ORDER BY COUNT ( * ) DESC ;",
			ExpectedPlaceHoldersCount: 1,
		},
		{
			Query:                     "INSERT INTO sbtest1 (id, k, c, pad) VALUES (4062, 72, '80700175623-243441', '76422972981-022')",
			DigestText:                "INSERT INTO `sbtest1` ( `id` , `k` , `c` , `pad` ) VALUES (...)",
			ExpectedFingerprint:       "INSERT INTO `sbtest1` ( `id` , `k` , `c` , `pad` ) VALUES (:1, :2, :3, :4)",
			ExpectedPlaceHoldersCount: 4,
		},
		{
			Query:                     "INSERT INTO sbtest1 (id, k, c, pad) VALUES (4062, 72, '80700175623-243441', '76422972981-022')",
			DigestText:                "INSERT INTO `sbtest1` ( `id` , `k` , `c` , `pad` ) VALUES (?+)",
			ExpectedFingerprint:       "INSERT INTO `sbtest1` ( `id` , `k` , `c` , `pad` ) VALUES (:1, :2, :3, :4)",
			ExpectedPlaceHoldersCount: 4,
		},
		{
			Query:                     "SELECT c FROM sbtest1 WHERE id BETWEEN 1 AND 100",
			DigestText:                "select c from sbtest1 where id between ? and ?",
			ExpectedFingerprint:       "select c from sbtest1 where id between :1 and :2",
			ExpectedPlaceHoldersCount: 2,
		},
	}

	for _, sql := range sqls {
		query, placeholdersCount, err := GetMySQLFingerprintPlaceholders(sql.Query, sql.DigestText)
		require.NoError(t, err)
		assert.Equal(t, sql.ExpectedFingerprint, query)
		assert.Equal(t, sql.ExpectedPlaceHoldersCount, placeholdersCount)
	}
}
