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

// Package queryparser provides functionality for queries fingerprint and placeholders parsing.
package queryparser

import (
	"fmt"
	"regexp"
	"strings"
)

var (
	allStringsRegexp           = regexp.MustCompile(`'.*?'|".*?"`)
	braceletsRegexp            = regexp.MustCompile(`\(.*?\)`)
	braceletsMultiformRegexp   = regexp.MustCompile(`\(\?\+\)|\(\.\.\.\)`)
	decimalsPlaceholdersRegexp = regexp.MustCompile(`:\d+`)
)

// GetMySQLFingerprintPlaceholders parse query and digest text and return fingerprint and placeholders count.
func GetMySQLFingerprintPlaceholders(query, digestText string) (string, uint32) {
	queryWithoutStrings := allStringsRegexp.ReplaceAllString(query, "")
	contents := make(map[int]string)
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

	return strings.TrimSpace(result), count
}

// GetMySQLFingerprintFromExplainFingerprint converts placeholders in fingerprint from our format (:1, :2 etc) into '?'
// to make it compatible with sql.Query functions.
func GetMySQLFingerprintFromExplainFingerprint(explainFingerprint string) string {
	return decimalsPlaceholdersRegexp.ReplaceAllString(explainFingerprint, "?")
}
