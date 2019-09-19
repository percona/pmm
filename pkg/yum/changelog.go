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

	"github.com/pkg/errors"
)

type changeLog struct {
	url string
}

var urlRE = regexp.MustCompile(`https?://\S+`) //nolint:gochecknoglobals

// parseChangeLog parses `yum update --changelog` stdout for a single version of a single package.
func parseChangeLog(lines []string) (*changeLog, error) {
	var foundChangeLog bool
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		if strings.HasPrefix(line, "ChangeLog for") {
			foundChangeLog = true
		}
		if !foundChangeLog {
			continue
		}
		if u := urlRE.FindString(line); u != "" {
			return &changeLog{
				url: u,
			}, nil
		}
	}

	return nil, errors.New("no URL found in changelog")
}
