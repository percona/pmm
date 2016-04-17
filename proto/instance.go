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
	"time"
)

type Subsystem struct {
	Id       uint
	ParentId uint
	Name     string // os, agent,  mysql
	Label    string // OS, Agent, MySQL
}

type Instance struct {
	Subsystem  string // Subsystem.Name
	ParentUUID string
	Id         uint   // internal ID for joining data tables
	UUID       string // primary ID, for accessing API
	Name       string // secondary ID, for human readability
	DSN        string // type-specific DSN, if any
	Distro     string
	Version    string
	Created    time.Time
	Deleted    time.Time
	Links      map[string]string `json:",omitempty"`
}
