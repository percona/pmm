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

package status

import (
	"fmt"

	"github.com/fatih/structs"
)

// Status converts stats into pct map status
type Status struct {
	stats interface{}
}

func New(stats interface{}) *Status {
	return &Status{
		stats: stats,
	}
}

// Map converts stats struct into a map
func (s *Status) Map() map[string]string {
	out := map[string]string{}
	for _, f := range structs.New(s.stats).Fields() {
		if f.IsZero() {
			continue
		}
		tag := f.Tag("name")
		if tag == "" {
			continue
		}
		v := fmt.Sprint(f.Value())
		if v == "" || v == `""` || v == "0" {
			continue
		}
		out[tag] = v
	}
	return out
}
