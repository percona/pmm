// pmm-admin
// Copyright (C) 2018 Percona LLC
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

package inventory

import (
	"fmt"
	"regexp"
	"strings"
)

func parseCustomLabels(labels string) (map[string]string, error) {
	if labels == "" {
		return map[string]string{}, nil
	}
	regex := regexp.MustCompile(`(\w+)=(\w+)`)
	result := make(map[string]string)
	parts := strings.Split(labels, ",")
	for _, part := range parts {
		if !regex.MatchString(part) {
			return nil, fmt.Errorf("wrong custom label format")
		}
		submatches := regex.FindStringSubmatch(part)
		result[submatches[1]] = submatches[2]
	}
	return result, nil
}
