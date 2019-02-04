// Package version provides exported variables that should be set during build to specify PMM version.
package version

import (
	"fmt"
	"regexp"
	"strconv"
)

var versionRE = regexp.MustCompile(`^(\d+)\.(\d+)\.(\d+)(.*)$`)

type ParsedVersion struct {
	Major int
	Minor int
	Patch int
	Rest  string
}

func Parse(version string) (ParsedVersion, error) {
	m := versionRE.FindStringSubmatch(version)
	if len(m) != 5 {
		return ParsedVersion{}, fmt.Errorf("failed to parse %q", version)
	}
	pv := ParsedVersion{Rest: m[4]}
	var err error
	if pv.Major, err = strconv.Atoi(m[1]); err != nil {
		return ParsedVersion{}, err
	}
	if pv.Minor, err = strconv.Atoi(m[2]); err != nil {
		return ParsedVersion{}, err
	}
	if pv.Patch, err = strconv.Atoi(m[3]); err != nil {
		return ParsedVersion{}, err
	}
	return pv, nil
}
