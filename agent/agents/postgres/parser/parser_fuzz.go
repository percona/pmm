// Copyright (C) 2023 Percona LLC
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

//go:build gofuzz
// +build gofuzz

// See https://github.com/dvyukov/go-fuzz

package parser

import (
	"fmt"
	"reflect"
	"unicode/utf8"
)

func init() {
	extractTablesRecover = false
}

func Fuzz(b []byte) int {
	// FIXME do we need this?
	if !utf8.Valid(b) {
		return -1
	}

	query := string(b)
	newT, newErr := ExtractTables(query)
	oldT, oldErr := extractTablesOld(query)

	var newErrS, oldErrS string
	if newErr != nil {
		newErrS = newErr.Error()
	}
	if oldErr != nil {
		oldErrS = oldErr.Error()
	}
	if newErrS != oldErrS {
		panic(fmt.Sprintf("Errors differ:\n\nnewErr = %+v\n\noldErr = %+v", newErr, oldErr))
	}

	if !reflect.DeepEqual(newT, oldT) {
		panic(fmt.Sprintf("%v != %v", newT, oldT))
	}

	if newErr == nil {
		return 1
	}

	return 0
}
