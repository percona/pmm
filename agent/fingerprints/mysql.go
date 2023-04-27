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
	"strings"
)

// GetMySQLFingerprintPlaceholders parse digest text and return fingerprint and placeholders count.
func GetMySQLFingerprintPlaceholders(digestText string) (string, uint32, error) {
	replacer := strings.NewReplacer("(?+)", "?", "(...)", "?")
	fingerprint := replacer.Replace(digestText)

	var count uint32
	for {
		i := strings.Index(fingerprint, "?")
		if i == -1 {
			break
		}

		count++
		fingerprint = strings.Replace(fingerprint, "?", fmt.Sprintf(":%d", count), 1)
	}

	return fingerprint, count, nil
}

func testo() (string, uint32, error) {
	query := "INSERT INTO sbtest1 (id, k, c, pad) VALUES ( 0, 561, 'y', 'x')"
	normalized := "insert into sbtest1 (id, k, c, pad) values(?+)"

	return fingerprint, count, nil
}
