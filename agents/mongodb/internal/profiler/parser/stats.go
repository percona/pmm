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

package parser

type stats struct {
	InDocs         int64  `name:"in-docs"`
	OkDocs         int64  `name:"ok-docs"`
	OutReports     int64  `name:"out-reports"`
	IntervalStart  string `name:"interval-start"`
	IntervalEnd    string `name:"interval-end"`
	ErrFingerprint int64  `name:"err-fingerprint"`
	ErrParse       int64  `name:"err-parse"`
	SkippedDocs    int64  `name:"skipped-docs"`
}
