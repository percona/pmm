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

package collector

type stats struct {
	In                     int64  `name:"in"`
	Out                    int64  `name:"out"`
	IteratorCreated        string `name:"iterator-created"`
	IteratorCounter        int64  `name:"iterator-counter"`
	IteratorRestartCounter int64  `name:"iterator-restart-counter"`
	IteratorErrLast        string `name:"iterator-err-last"`
	IteratorErrCounter     int64  `name:"iterator-err-counter"`
	IteratorTimeout        int64  `name:"iterator-timeout"`
}
