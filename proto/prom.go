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

type Exporter struct {
	Name         string   `yaml:"name"`
	Alias        string   `yaml:"alias,omitempty"`
	Port         string   `yaml:"port"`
	Args         []string `yaml:"args"`
	InstanceUUID string   `yaml:"instance_uuid,omitempty"`
}

var HostTypes = []string{"os", "mysql"}

type Host struct {
	Alias   string
	Address string
}

type Endpoint struct {
	Targets []string
	Labels  map[string]string
}
