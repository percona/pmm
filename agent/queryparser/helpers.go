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

// Package queryparser provides functionality for queries parsing.
package queryparser

import (
	"bufio"
	"fmt"
	"regexp"
	"strings"
)

func parseMySQLComments(q string) (map[string]bool, error) {
	// sqlparser.ExtractMysqlComment(q) doesnt work properly
	// input: SELECT * FROM people /*! bla bla */ WHERE name = 'john'
	// output: ECT * FROM people /*! bla bla */ WHERE name = 'joh
	multiline, err := regexp.Compile("(?s)\\/\\*(.*?)\\*\\/")
	if err != nil {
		return nil, err
	}
	// space := regexp.MustCompile(`\s+`)
	// if err != nil {
	// 	return nil, err
	// }

	// comments using comment as a key to avoid duplicates
	comments := make(map[string]bool)
	for _, v := range multiline.FindAllStringSubmatch(q, -1) {
		if len(v) < 2 {
			continue
		}

		value := strings.ReplaceAll(v[1], "\n", "")
		value = strings.ReplaceAll(value, "\t", "")
		value = strings.TrimSpace(value)
		// handle all mutations of multiline comment
		// // /*! and /*+
		value = strings.TrimLeft(value, "!")
		value = strings.TrimLeft(value, "+")
		value = strings.TrimLeft(value, " ")
		value = strings.TrimRight(value, " ")

		comments[value] = true
	}

	hashComments, err := parseMySQLSinglelineComments(q, "#")
	if err != nil {
		return nil, err
	}
	for c := range hashComments {
		comments[c] = true
	}

	dashComments, err := parseMySQLSinglelineComments(q, "--")
	if err != nil {
		return nil, err
	}
	for c := range dashComments {
		comments[c] = true
	}

	return comments, nil
}

func parseMySQLSinglelineComments(q, startChar string) (map[string]bool, error) {
	r, err := regexp.Compile(fmt.Sprintf("%s.*", startChar))
	if err != nil {
		return nil, err
	}

	// comments using comment as a key to avoid duplicates
	comments := make(map[string]bool)
	lines, err := stringToLines(q)
	if err != nil {
		return nil, err
	}
	for _, l := range lines {
		for _, v := range r.FindStringSubmatch(l) {

			comments[strings.TrimLeft(v, fmt.Sprintf("%s ", startChar))] = true
		}
	}

	return comments, nil
}

func stringToLines(s string) (lines []string, err error) {
	scanner := bufio.NewScanner(strings.NewReader(s))
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	err = scanner.Err()

	return
}
