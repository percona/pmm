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

package proto

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"
)

type NullString struct {
	sql.NullString
}

func (n *NullString) MarshalJSON() (b []byte, err error) {
	if !n.Valid {
		return []byte("null"), nil
	}
	return json.Marshal(n.String)
}

func (n *NullString) UnmarshalJSON(b []byte) error {
	if bytes.Equal(b, []byte("null")) {
		n.String = ""
		n.Valid = false
		return nil
	}
	err := json.Unmarshal(b, &n.String)
	if err != nil {
		return err
	}
	n.Valid = true
	return nil
}

func (n NullString) Equal(u NullString) bool {
	if !n.Valid && !u.Valid {
		return true // both are null
	} else if (n.Valid && !u.Valid) || (!n.Valid && u.Valid) {
		return false // only one isn't null
	} else {
		return n.String == u.String
	}
}

// --------------------------------------------------------------------------

type NullInt64 struct {
	sql.NullInt64
}

func (n *NullInt64) MarshalJSON() (b []byte, err error) {
	if !n.Valid {
		return []byte("null"), nil
	}
	return json.Marshal(n.Int64)
}

func (n *NullInt64) UnmarshalJSON(b []byte) error {
	if bytes.Equal(b, []byte("null")) {
		n.Int64 = 0
		n.Valid = false
		return nil
	}
	err := json.Unmarshal(b, &n.Int64)
	if err != nil {
		return err
	}
	n.Valid = true
	return nil
}

func (n NullInt64) Equal(u NullInt64) bool {
	if !n.Valid && !u.Valid {
		return true // both are null
	} else if (n.Valid && !u.Valid) || (!n.Valid && u.Valid) {
		return false // only one isn't null
	} else {
		return n.Int64 == u.Int64
	}
}

// --------------------------------------------------------------------------

type NullFloat64 struct {
	sql.NullFloat64
}

func (n NullFloat64) MarshalJSON() (b []byte, err error) {
	if !n.Valid {
		return []byte("null"), nil
	}
	return json.Marshal(n.Float64)
}

func (n *NullFloat64) UnmarshalJSON(b []byte) error {
	if bytes.Equal(b, []byte("null")) {
		n.Float64 = 0
		n.Valid = false
		return nil
	}
	err := json.Unmarshal(b, &n.Float64)
	if err != nil {
		return err
	}
	n.Valid = true
	return nil
}

func (n NullFloat64) Equal(u NullFloat64) bool {
	if !n.Valid && !u.Valid {
		return true // both are null
	} else if (n.Valid && !u.Valid) || (!n.Valid && u.Valid) {
		return false // only one isn't null
	} else {
		// we only need microsecond accuracy
		nval := fmt.Sprintf("%.6f", n.Float64)
		uval := fmt.Sprintf("%.6f", u.Float64)
		return nval == uval
	}
}

// --------------------------------------------------------------------------

type NullTime struct {
	time.Time
}

func (n *NullTime) Scan(src interface{}) error {
	if src != nil {
		n.Time = src.(time.Time)
	}
	return nil
}
