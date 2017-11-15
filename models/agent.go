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

package models

//reform:agents
type Agent struct {
	ID           int32  `reform:"id,pk"`
	Type         string `reform:"type"`
	RunsOnNodeID int32  `reform:"runs_on_node_id"`
}

//reform:agents
type MySQLdExporter struct {
	ID           int32  `reform:"id,pk"`
	Type         string `reform:"type"`
	RunsOnNodeID int32  `reform:"runs_on_node_id"`

	Login    string `reform:"login"`
	Password string `reform:"password"`
}
