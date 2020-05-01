package version

import (
	"fmt"
	"regexp"
	"strconv"
)

var versionRE = regexp.MustCompile(`^(\d+)\.(\d+)\.(\d+)(.*)$`)

// Parsed represents a SemVer-like version information.
//
// int64 are used instead of int for compatibility with Starlark.
type Parsed struct {
	Major int64
	Minor int64
	Patch int64
	Rest  string
	Num   int64 // MMmmpp
}

// Parse parses version information from given string.
func Parse(s string) (*Parsed, error) {
	m := versionRE.FindStringSubmatch(s)
	if len(m) != 5 {
		return nil, fmt.Errorf("failed to parse %q", s)
	}

	res := &Parsed{Rest: m[4]}
	var err error
	if res.Major, err = strconv.ParseInt(m[1], 10, 64); err != nil {
		return nil, err
	}
	if res.Minor, err = strconv.ParseInt(m[2], 10, 64); err != nil {
		return nil, err
	}
	if res.Patch, err = strconv.ParseInt(m[3], 10, 64); err != nil {
		return nil, err
	}

	res.Num = res.Major*10000 + res.Minor*100 + res.Patch
	return res, nil
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
