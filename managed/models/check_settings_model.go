// Copyright (C) 2023 Percona LLC
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

package models

//go:generate ../../bin/reform

// Interval represents check execution interval.
type Interval string

// Available check execution intervals.
const (
	Standard Interval = "standard"
	Frequent Interval = "frequent"
	Rare     Interval = "rare"
)

// CheckSettings represents any changes to an Advisor check loaded in pmm-managed.
//
//reform:check_settings
type CheckSettings struct {
	Name     string   `reform:"name,pk"`
	Interval Interval `reform:"interval"`
}
