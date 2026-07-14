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

// Package yum provides functionality for yum dependency manager.
package yum

import (
	"os/exec"
	"regexp"
	"strings"
	"time"

	"github.com/pkg/errors"
)

// parseInfo parses `yum info` stdout for a single version of a single package.
// Also used to parse `yum repoinfo`.
func parseInfo(lines []string, firstKey string) (map[string]string, error) {
	res := make(map[string]string)
	var prevKey string
	var keyFound bool
	for _, line := range lines {
		// separate progress output from data
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		if key == firstKey {
			// sanity check that we do not try to parse multiple packages
			if keyFound {
				return res, errors.New("second `Name` encountered")
			}
			keyFound = true
		}
		if !keyFound {
			continue
		}

		// parse data while handling multiline values
		value := strings.TrimSpace(parts[1])
		if key == "" {
			if prevKey != "" {
				res[prevKey] += " " + value
			}
			continue
		}
		res[key] = value
		prevKey = key
	}
	return res, nil
}

func parseInfoTime(s string) (time.Time, error) {
	layout := "Mon 2 Jan 2006 15:04:05 PM UTC" // layout for EL9, default
	v, err := getRHELVersion()
	if err == nil && v == "7" {
		layout = "Mon Jan 2 15:04:05 2006" // change the layout for EL7
	}
	return time.Parse(layout, s)
}

// fullVersion returns full (ugly) package version.
func fullVersion(info map[string]string) string {
	var res string
	if e := info["Epoch"]; e != "" {
		res = e + ":"
	}
	res += info["Version"]
	res += "-" + info["Release"]
	return res
}

// niceVersion returns nice user-visible package version.
func niceVersion(info map[string]string) string {
	// cut suffixes from full release
	release := info["Release"]
	for _, re := range []*regexp.Regexp{
		regexp.MustCompile(`^(.*)\.el\d+$`),       // el7 suffix
		regexp.MustCompile(`^(.*)\.[0-9a-f]{7}$`), // abbreviated commit suffix
		regexp.MustCompile(`^(.*)\.\d{10}$`),      // timestamp suffix
	} {
		release = re.ReplaceAllString(release, "$1")
	}

	// if there is more than just release digits (like `9.beta5` or `18.rc4`), return them;
	// return only version otherwise.
	if !regexp.MustCompile(`^\d+$`).MatchString(release) {
		return info["Version"] + "-" + release
	}
	return info["Version"]
}

func getRHELVersion() (string, error) {
	raw, err := exec.Command("rpm", "--eval", "%{rhel}").Output()
	if err != nil {
		return "", errors.Wrap(err, "couldn't get RHEL version")
	}

	return strings.TrimSpace(string(raw)), nil
}
