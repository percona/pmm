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

package parser

type stats struct {
	InDocs         int64  `name:"in-docs"`
	OkDocs         int64  `name:"ok-docs"`
	OutReports     int64  `name:"out-reports"`
	IntervalStart  string `name:"interval-start"`
	IntervalEnd    string `name:"interval-end"`
	ErrFingerprint int64  `name:"err-fingerprint"`
	ErrParse       int64  `name:"err-parse"`
	SkippedDocs    int64  `name:"skipped-docs"`
}
