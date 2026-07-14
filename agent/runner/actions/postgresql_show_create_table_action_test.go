// Copyright (C) 2023 Percona LLC
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

package actions

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/percona/pmm/agent/utils/tests"
	"github.com/percona/pmm/api/agentpb"
)

func TestPostgreSQLShowCreateTable(t *testing.T) {
	t.Parallel()

	dsn := tests.GetTestPostgreSQLDSN(t)
	db := tests.OpenTestPostgreSQL(t)
	t.Cleanup(func() { db.Close() }) //nolint:errcheck

	t.Run("With Schema Name", func(t *testing.T) {
		t.Parallel()
		params := &agentpb.StartActionRequest_PostgreSQLShowCreateTableParams{
			Dsn:   dsn,
			Table: "public.country",
		}
		a, err := NewPostgreSQLShowCreateTableAction("", 0, params, os.TempDir())
		require.NoError(t, err)

		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()

		b, err := a.Run(ctx)
		require.NoError(t, err)

		expected := `Table "public.country"
Column         |Type          |Collation |Nullable |Default |Storage  |Stats target |Description
code           |character(3)  |          |not null |        |extended |             |
name           |text          |          |not null |        |extended |             |
continent      |text          |          |not null |        |extended |             |
region         |text          |          |not null |        |extended |             |
surfacearea    |real          |          |not null |        |plain    |             |
indepyear      |smallint      |          |         |        |plain    |             |
population     |integer       |          |not null |        |plain    |             |
lifeexpectancy |real          |          |         |        |plain    |             |
gnp            |numeric(10,2) |          |         |        |main     |             |
gnpold         |numeric(10,2) |          |         |        |main     |             |
localname      |text          |          |not null |        |extended |             |
governmentform |text          |          |not null |        |extended |             |
headofstate    |text          |          |         |        |extended |             |
capital        |integer       |          |         |        |plain    |             |
code2          |character(2)  |          |not null |        |extended |             |
Indexes:
	"country_pkey" PRIMARY KEY, btree (code)
Check constraints:
	"country_continent_check" CHECK (continent = 'Asia'::text OR continent = 'Europe'::text OR continent = 'North America'::text OR continent = 'Africa'::text OR continent = 'Oceania'::text OR continent = 'Antarctica'::text OR continent = 'South America'::text)
Foreign-key constraints:
	"country_capital_fkey" FOREIGN KEY (capital) REFERENCES city(id)
Referenced by:
	TABLE "countrylanguage" CONSTRAINT "countrylanguage_countrycode_fkey" FOREIGN KEY (countrycode) REFERENCES country(code)
`

		assert.Equal(t, expected, string(b))
	})

	t.Run("Without constraints", func(t *testing.T) {
		t.Parallel()
		params := &agentpb.StartActionRequest_PostgreSQLShowCreateTableParams{
			Dsn:   dsn,
			Table: "city",
		}
		a, err := NewPostgreSQLShowCreateTableAction("", 0, params, os.TempDir())
		require.NoError(t, err)

		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()

		b, err := a.Run(ctx)
		require.NoError(t, err)

		expected := `Table "public.city"
Column      |Type         |Collation |Nullable |Default |Storage  |Stats target |Description
id          |integer      |          |not null |        |plain    |             |
name        |text         |          |not null |        |extended |             |
countrycode |character(3) |          |not null |        |extended |             |
district    |text         |          |not null |        |extended |             |
population  |integer      |          |not null |        |plain    |             |
Indexes:
	"city_pkey" PRIMARY KEY, btree (id)
Referenced by:
	TABLE "country" CONSTRAINT "country_capital_fkey" FOREIGN KEY (capital) REFERENCES city(id)
`

		assert.Equal(t, expected, string(b))
	})

	t.Run("Without references", func(t *testing.T) {
		t.Parallel()
		params := &agentpb.StartActionRequest_PostgreSQLShowCreateTableParams{
			Dsn:   dsn,
			Table: "countrylanguage",
		}
		a, err := NewPostgreSQLShowCreateTableAction("", 0, params, os.TempDir())
		require.NoError(t, err)

		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()

		b, err := a.Run(ctx)
		require.NoError(t, err)

		expected := `Table "public.countrylanguage"
Column      |Type         |Collation |Nullable |Default |Storage  |Stats target |Description
countrycode |character(3) |          |not null |        |extended |             |
language    |text         |          |not null |        |extended |             |
isofficial  |boolean      |          |not null |        |plain    |             |
percentage  |real         |          |not null |        |plain    |             |
Indexes:
	"countrylanguage_pkey" PRIMARY KEY, btree (countrycode, language)
Foreign-key constraints:
	"countrylanguage_countrycode_fkey" FOREIGN KEY (countrycode) REFERENCES country(code)
`

		assert.Equal(t, expected, string(b))
	})

	t.Run("LittleBobbyTables", func(t *testing.T) {
		t.Parallel()
		params := &agentpb.StartActionRequest_PostgreSQLShowCreateTableParams{
			Dsn:   dsn,
			Table: `city; DROP TABLE city; --`,
		}
		a, err := NewPostgreSQLShowCreateTableAction("", 0, params, os.TempDir())
		require.NoError(t, err)

		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()

		_, err = a.Run(ctx)
		expected := "Table not found: sql: no rows in result set"
		assert.EqualError(t, err, expected)

		var count int
		err = db.QueryRow("SELECT COUNT(*) FROM city").Scan(&count)
		require.NoError(t, err)
		assert.Equal(t, 4079, count)
	})
}
