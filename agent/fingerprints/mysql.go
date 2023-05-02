// Copyright 2019 Percona LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//  http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package fingerprints provides functionality for queries fingerprint and placeholders parsing.
package fingerprints

import (
	"fmt"
	"regexp"
	"strings"
)

// GetMySQLFingerprintPlaceholders parse query and digest text and return fingerprint and placeholders count.
func GetMySQLFingerprintPlaceholders(query, digestText string) (string, uint32, error) {
	r, err := regexp.Compile(`'.*?'|".*?"`)
	if err != nil {
		return "", 0, err
	}
	res := r.ReplaceAllString(query, "")

	r2, err := regexp.Compile(`\(.*?\)`)
	if err != nil {
		return "", 0, err
	}

	new := map[int]string{}
	rs := r2.FindAllString(res, -1)
	for k, v := range rs {
		count := strings.Count(v, ",")
		new[k] = fmt.Sprintf("(%s?)", strings.Repeat("?, ", count))
	}
	r3, err := regexp.Compile(`\(\?\+\)|\(\.\.\.\)`)
	if err != nil {
		return "", 0, err
	}

	i := 0
	rs2 := r3.ReplaceAllStringFunc(digestText, func(s string) string {
		val := new[i]
		i++
		return val
	})

	var count uint32
	for {
		i := strings.Index(rs2, "?")
		if i == -1 {
			break
		}

		count++
		rs2 = strings.Replace(rs2, "?", fmt.Sprintf(":%d", count), 1)
	}

	return rs2, uint32(count), nil
}
