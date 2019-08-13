// pmm-agent
// Copyright (C) 2018 Percona LLC
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

package actions

import (
	"context"
	"testing"
	"time"

	"github.com/percona/pmm/api/agentpb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/percona/pmm-agent/utils/tests"
)

func TestPostgreSQLShowCreateTable(t *testing.T) {
	t.Parallel()

	dsn := tests.GetTestPostgreSQLDSN(t)
	db := tests.OpenTestPostgreSQL(t)
	defer db.Close() //nolint:errcheck

	t.Run("Default", func(t *testing.T) {
		params := &agentpb.StartActionRequest_PostgreSQLShowCreateTableParams{
			Dsn:   dsn,
			Table: "country",
		}
		a := NewPostgreSQLShowCreateTableAction("", params)
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
		params := &agentpb.StartActionRequest_PostgreSQLShowCreateTableParams{
			Dsn:   dsn,
			Table: "city",
		}
		a := NewPostgreSQLShowCreateTableAction("", params)
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
		params := &agentpb.StartActionRequest_PostgreSQLShowCreateTableParams{
			Dsn:   dsn,
			Table: "countrylanguage",
		}
		a := NewPostgreSQLShowCreateTableAction("", params)
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
		params := &agentpb.StartActionRequest_PostgreSQLShowCreateTableParams{
			Dsn:   dsn,
			Table: `city; DROP TABLE city; --`,
		}
		a := NewPostgreSQLShowCreateTableAction("", params)
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()

		_, err := a.Run(ctx)
		expected := "Table not found: sql: no rows in result set"
		assert.EqualError(t, err, expected)

		var count int
		err = db.QueryRow("SELECT COUNT(*) FROM city").Scan(&count)
		require.NoError(t, err)
		assert.Equal(t, 4079, count)
	})
}
