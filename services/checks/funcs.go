// pmm-managed
// Copyright (C) 2017 Percona LLC
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

package checks

import (
	"github.com/percona-platform/saas/pkg/starlark"
	"github.com/percona/pmm/version"
	"github.com/pkg/errors"
)

// GetFuncsForVersion returns predefined functions for specified check version.
func GetFuncsForVersion(version uint32) (map[string]starlark.GoFunc, error) {
	switch version {
	case 1:
		return map[string]starlark.GoFunc{
			"parse_version":      parseVersion,
			"format_version_num": formatVersionNum,
		}, nil
	default:
		return nil, errors.Errorf("unsupported check version: %d", version)
	}
}

// parseVersion accepts a single string argument (version), and returns map[string]interface{}
// with keys: major, minor, patch (int64), num (MMmmpp, int64), and rest (string).
func parseVersion(args ...interface{}) (interface{}, error) {
	if l := len(args); l != 1 {
		return nil, errors.Errorf("expected 1 argument, got %d", l)
	}

	s, ok := args[0].(string)
	if !ok {
		return nil, errors.Errorf("expected string argument, got %[1]T (%[1]v)", args[0])
	}

	p, err := version.Parse(s)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"major": int64(p.Major),
		"minor": int64(p.Minor),
		"patch": int64(p.Patch),
		"rest":  p.Rest,
		"num":   int64(p.Num),
	}, nil
}

// formatVersionNum accepts a single int64 argument (version num MMmmpp), and returns
// MM.mm.pp as a string.
func formatVersionNum(args ...interface{}) (interface{}, error) {
	if l := len(args); l != 1 {
		return nil, errors.Errorf("expected 1 argument, got %d", l)
	}

	num, ok := args[0].(int64)
	if !ok {
		return nil, errors.Errorf("expected int64 argument, got %[1]T (%[1]v)", args[0])
	}

	p := &version.Parsed{
		Major: int(num / 10000),
		Minor: int(num / 100 % 100),
		Patch: int(num % 100),
	}
	return p.String(), nil
}
