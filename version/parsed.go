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

package version

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/pkg/errors"
)

var (
	versionRE = regexp.MustCompile(`^(\d+)\.(\d+)\.(\d+)(.*)$`)
	fetchRest = regexp.MustCompile(`^-(\d+)-?(.*)$`)
)

// Parsed represents a SemVer-like version information.
type Parsed struct {
	Major   int
	Minor   int
	Patch   int
	Rest    string
	Num     int // MMmmpp
	NumRest int
}

// Parse parses version information from given string.
func Parse(s string) (*Parsed, error) {
	m := versionRE.FindStringSubmatch(s)
	if len(m) != 5 {
		return nil, errors.Errorf("failed to parse %q", s)
	}

	res := &Parsed{Rest: m[4]}
	var err error
	if res.Major, err = strconv.Atoi(m[1]); err != nil {
		return nil, err
	}
	if res.Minor, err = strconv.Atoi(m[2]); err != nil {
		return nil, err
	}
	if res.Patch, err = strconv.Atoi(m[3]); err != nil {
		return nil, err
	}

	r := fetchRest.FindStringSubmatch(res.Rest)
	if len(r) != 0 {
		if res.NumRest, err = strconv.Atoi(r[1]); err != nil {
			return nil, err
		}
	}

	res.Num = res.Major*10000 + res.Minor*100 + res.Patch
	return res, nil
}

// MustParse is like Parse but panics if given string cannot be parsed.
// It simplifies safe initialization of global variables holding parsed versions.
func MustParse(s string) *Parsed {
	p, err := Parse(s)
	if err != nil {
		panic(err)
	}
	return p
}

// String returns original string representation of version information.
func (p *Parsed) String() string {
	return fmt.Sprintf("%d.%d.%d%s", p.Major, p.Minor, p.Patch, p.Rest)
}

// Less returns true if this (left) Info is less than given argument (right).
func (p *Parsed) Less(right *Parsed) bool {
	if p.Num != right.Num {
		return p.Num < right.Num
	}

	switch {
	case p.Rest == "" && right.Rest == "": // versions are equal, "less" is false
		return false
	case p.Rest == "" && right.Rest != "": // v2.0.0 > v2.0.0-beta4
		return false
	case p.Rest != "" && right.Rest == "": // v2.0.0-beta4 < v2.0.0
		return true
	}

	return p.Rest < right.Rest
}

func (p *Parsed) UnmarshalJSON(b []byte) error {
	s := string(b)
	s = strings.Trim(s, `"`)
	parsed, err := Parse(s)
	if err != nil {
		return err
	}
	*p = *parsed
	return nil
}
