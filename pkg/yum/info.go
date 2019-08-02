// pmm-update
// Copyright (C) 2019 Percona LLC
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

package yum

import (
	"regexp"
	"strings"
	"time"

	"github.com/pkg/errors"
)

// parseInfo parses `yum info` stdout for a single version of a single package.
func parseInfo(lines []string) (map[string]string, error) {
	res := make(map[string]string)
	var prevKey string
	var nameFound bool
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
		if key == "Name" {
			// sanity check that we do not try to parse multiple packages
			if nameFound {
				return res, errors.New("second `Name` encountered")
			}
			nameFound = true
		}
		if !nameFound {
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
	return time.Parse("Mon Jan 2 15:04:05 2006", s)
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
	// cut suffixes and the first digits section from full release
	release := info["Release"]
	for _, re := range []*regexp.Regexp{
		regexp.MustCompile(`^(.*)\.el\d+$`),       // el7 suffix
		regexp.MustCompile(`^(.*)\.[0-9a-f]{7}$`), // abbriviated commit suffix
		regexp.MustCompile(`^(.*)\.\d{10}$`),      // timestamp suffix
		regexp.MustCompile(`^\d+\.(.*)$`),         // first digits section
	} {
		release = re.ReplaceAllString(release, "$1")
	}

	return info["Version"] + "-" + release
}
