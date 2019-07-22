// Package version provides helpers for working with versions and build info.
package version

import (
	"fmt"
	"regexp"
	"strconv"
)

var (
	gitDescribeRE = regexp.MustCompile(`^v?(\d+)\.(\d+)\.(\d+)(.*)$`)
	rpmRE         = regexp.MustCompile(`^(\d+)\.(\d+)\.(\d+)-(\d+)(.*)$`)
)

// GitDescribeInfo contains information about PMM component produced by `git describe` command.
// It is embedded into Go components by `make release`.
type GitDescribeInfo struct {
	Major int
	Minor int
	Patch int
	Rest  string // alpha/beta, number of commits, abbreviated commit, dirty
}

// ParseGitDescribeInfo parses GitDescribeInfo from given string.
func ParseGitDescribeInfo(s string) (*GitDescribeInfo, error) {
	m := gitDescribeRE.FindStringSubmatch(s)
	if len(m) != 5 {
		return nil, fmt.Errorf("failed to parse %q", s)
	}

	info := &GitDescribeInfo{Rest: m[4]}
	var err error
	if info.Major, err = strconv.Atoi(m[1]); err != nil {
		return nil, err
	}
	if info.Minor, err = strconv.Atoi(m[2]); err != nil {
		return nil, err
	}
	if info.Patch, err = strconv.Atoi(m[3]); err != nil {
		return nil, err
	}
	return info, nil
}

func (i *GitDescribeInfo) String() string {
	res := fmt.Sprintf("v%d.%d.%d", i.Major, i.Minor, i.Patch)
	if i.Rest != "" {
		res += i.Rest
	}
	return res
}

// Less returns true if this (left) GitDescribeInfo is less than given argument (right).
func (i *GitDescribeInfo) Less(right *GitDescribeInfo) bool {
	if i.Major != right.Major {
		return i.Major < right.Major
	}
	if i.Minor != right.Minor {
		return i.Minor < right.Minor
	}
	if i.Patch != right.Patch {
		return i.Patch < right.Patch
	}

	switch {
	case i.Rest == "" && right.Rest == "": // versions are equal, "less" is false
		return false
	case i.Rest == "" && right.Rest != "": // v2.0.0 > v2.0.0-beta4
		return false
	case i.Rest != "" && right.Rest == "": // v2.0.0-beta4 < v2.0.0
		return true
	}

	return i.Rest < right.Rest
}

// RPMInfo contains information about PMM component's RPM package.
type RPMInfo struct {
	Major   int
	Minor   int
	Patch   int
	Release int
	Rest    string // alpha/beta, build_timestamp, shortcommit, dist
}

// ParseRPMInfo parses RPMInfo from given string.
func ParseRPMInfo(s string) (*RPMInfo, error) {
	m := rpmRE.FindStringSubmatch(s)
	if len(m) != 6 {
		return nil, fmt.Errorf("failed to parse %q", s)
	}

	info := &RPMInfo{Rest: m[5]}
	var err error
	if info.Major, err = strconv.Atoi(m[1]); err != nil {
		return nil, err
	}
	if info.Minor, err = strconv.Atoi(m[2]); err != nil {
		return nil, err
	}
	if info.Patch, err = strconv.Atoi(m[3]); err != nil {
		return nil, err
	}
	if info.Release, err = strconv.Atoi(m[4]); err != nil {
		return nil, err
	}
	return info, nil
}

func (i *RPMInfo) String() string {
	res := fmt.Sprintf("%d.%d.%d-%d", i.Major, i.Minor, i.Patch, i.Release)
	if i.Rest != "" {
		res += i.Rest
	}
	return res
}

// Less returns true if this (left) RPMInfo is less than given argument (right).
func (i *RPMInfo) Less(right *RPMInfo) bool {
	if i.Major != right.Major {
		return i.Major < right.Major
	}
	if i.Minor != right.Minor {
		return i.Minor < right.Minor
	}
	if i.Patch != right.Patch {
		return i.Patch < right.Patch
	}
	if i.Release != right.Release {
		return i.Release < right.Release
	}

	switch {
	case i.Rest == "" && right.Rest == "": // versions are equal, "less" is false
		return false
	case i.Rest == "" && right.Rest != "": // 2.0.0 > 2.0.0-beta4
		return false
	case i.Rest != "" && right.Rest == "": // 2.0.0-beta4 < 2.0.0
		return true
	}

	return i.Rest < right.Rest
}
