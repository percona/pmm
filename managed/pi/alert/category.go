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

package alert

import "fmt"

// Supported template categories.
const (
	CategoryUnknown    = Category("unknown")
	CategoryPMM        = Category("pmm")
	CategoryMongoDB    = Category("mongodb")
	CategoryMySQL      = Category("mysql")
	CategoryNode       = Category("node")
	CategoryPostgreSQL = Category("postgresql")
	CategoryProxySQL   = Category("proxysql")
	CategoryValkey     = Category("valkey")
	CategoryHAProxy    = Category("haproxy")
)

// Category represents the technology an alert template applies to.
type Category string

// Validate returns error in case of invalid category.
func (c Category) Validate() error {
	switch c {
	case "": // can be empty, treated as CategoryUnknown
		return nil
	case CategoryUnknown:
		return nil
	case CategoryPMM:
		return nil
	case CategoryMongoDB:
		return nil
	case CategoryMySQL:
		return nil
	case CategoryNode:
		return nil
	case CategoryPostgreSQL:
		return nil
	case CategoryProxySQL:
		return nil
	case CategoryValkey:
		return nil
	case CategoryHAProxy:
		return nil
	}

	// do not add `default:` to make exhaustive linter do its job

	return fmt.Errorf("unhandled template category '%s'", string(c))
}

// OrDefault returns the category, falling back to CategoryUnknown when absent.
// It does not mutate the receiver: an absent category must stay absent in the
// template's YAML representation.
func (c Category) OrDefault() Category {
	if c == "" {
		return CategoryUnknown
	}

	return c
}
