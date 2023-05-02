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
	"sync"
)

var (
	removeStringsRegexp *regexp.Regexp
	errRemoveStrings    error

	braceletsRegexp *regexp.Regexp
	errBracelets    error

	braceletsMultiformRegexp *regexp.Regexp
	errBraceletsMultiform    error

	once sync.Once
)

// GetMySQLFingerprintPlaceholders parse query and digest text and return fingerprint and placeholders count.
func GetMySQLFingerprintPlaceholders(query, digestText string) (string, uint32, error) {
	once.Do(func() {
		removeStringsRegexp, errRemoveStrings = regexp.Compile(`'.*?'|".*?"`)
		braceletsRegexp, errBracelets = regexp.Compile(`\(.*?\)`)
		braceletsMultiformRegexp, errBraceletsMultiform = regexp.Compile(`\(\?\+\)|\(\.\.\.\)`)
	})
	if errRemoveStrings != nil {
		return "", 0, errRemoveStrings
	}
	if errBracelets != nil {
		return "", 0, errBracelets
	}
	if errBraceletsMultiform != nil {
		return "", 0, errBraceletsMultiform
	}

	queryWithoutStrings := removeStringsRegexp.ReplaceAllString(query, "")
	contents := map[int]string{}
	bracelets := braceletsRegexp.FindAllString(queryWithoutStrings, -1)
	for k, v := range bracelets {
		count := strings.Count(v, ",")
		contents[k] = fmt.Sprintf("(%s?)", strings.Repeat("?, ", count))
	}

	i := 0
	result := braceletsMultiformRegexp.ReplaceAllStringFunc(digestText, func(s string) string {
		c := contents[i]
		i++
		return c
	})

	var count uint32
	for {
		index := strings.Index(result, "?")
		if index == -1 {
			break
		}

		count++
		result = strings.Replace(result, "?", fmt.Sprintf(":%d", count), 1)
	}

	return result, uint32(count), nil
}
