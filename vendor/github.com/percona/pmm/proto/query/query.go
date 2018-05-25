/*
   Copyright (c) 2016, Percona LLC and/or its affiliates. All rights reserved.

   This program is free software: you can redistribute it and/or modify
   it under the terms of the GNU Affero General Public License as published by
   the Free Software Foundation, either version 3 of the License, or
   (at your option) any later version.

   This program is distributed in the hope that it will be useful,
   but WITHOUT ANY WARRANTY; without even the implied warranty of
   MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
   GNU Affero General Public License for more details.

   You should have received a copy of the GNU Affero General Public License
   along with this program.  If not, see <http://www.gnu.org/licenses/>
*/

package query

import (
	"fmt"
	"time"
)

type Query struct {
	Id          string // 9C8DEE410FA0E0C8
	Abstract    string // SELECT tbl1
	Fingerprint string // select col from tbl1 where id=?
	Tables      []Table
	FirstSeen   time.Time
	LastSeen    time.Time
	Status      string
}

type Table struct {
	Db    string
	Table string
}

func (t Table) String() string {
	if t.Db == "" {
		return t.Table
	}
	return fmt.Sprintf("%s.%s", t.Db, t.Table)
}

type Example struct {
	QueryId      string // Query.Id
	InstanceUUID string // Instance.UUID
	Period       time.Time
	Ts           time.Time
	Db           string
	QueryTime    float64
	Query        string
	Size         int // Original size of the Query, before any truncation.
}
