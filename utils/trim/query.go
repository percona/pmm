// pmm-agent
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

package trim

var maxQueryLength = 1024

// Query limits passed query string to 1024 Unicode runes, trimming it if necessary.
func Query(query string) string {
	runes := []rune(query)
	if len(runes) <= maxQueryLength {
		return string(runes)
	}
	return string(append(runes[:maxQueryLength-4], ' ', '.', '.', '.'))
}
