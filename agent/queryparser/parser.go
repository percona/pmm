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
	"sync"
)

var (
	allStringsRegexp *regexp.Regexp
	errAllStrings    error

	braceletsRegexp *regexp.Regexp
	errBracelets    error

	braceletsMultiformRegexp *regexp.Regexp
	errBraceletsMultiform    error

	decimalsPlaceholdersRegexp *regexp.Regexp
	errDecimalsPlaceholders    error

	once sync.Once
)

func prepareRegexps() error {
	once.Do(func() {
		allStringsRegexp, errAllStrings = regexp.Compile(`'.*?'|".*?"`)
		braceletsRegexp, errBracelets = regexp.Compile(`\(.*?\)`)
		braceletsMultiformRegexp, errBraceletsMultiform = regexp.Compile(`\(\?\+\)|\(\.\.\.\)`)
		decimalsPlaceholdersRegexp, errDecimalsPlaceholders = regexp.Compile(`:\d+`)
	})
	if errAllStrings != nil {
		return errAllStrings
	}
	if errBracelets != nil {
		return errBracelets
	}
	if errBraceletsMultiform != nil {
		return errBraceletsMultiform
	}
	if errDecimalsPlaceholders != nil {
		return errDecimalsPlaceholders
	}

	return nil
}

// GetMySQLFingerprintPlaceholders parse query and digest text and return fingerprint and placeholders count.
func GetMySQLFingerprintPlaceholders(query, digestText string) (string, uint32, error) {
	err := prepareRegexps()
	if err != nil {
		return "", 0, err
	}

	queryWithoutStrings := allStringsRegexp.ReplaceAllString(query, "")
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

	return strings.TrimSpace(result), count, nil
}

// GetMySQLFingerprintFromExplainFingerprint convert placeholders in fingerprint from our format (:1, :2 etc) into ?
// to make it compatible with sql.Query functions.
func GetMySQLFingerprintFromExplainFingerprint(explainFingerprint string) (string, error) {
	err := prepareRegexps()
	if err != nil {
		return "", err
	}

	return decimalsPlaceholdersRegexp.ReplaceAllString(explainFingerprint, "?"), nil
}
