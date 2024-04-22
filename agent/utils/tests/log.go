// Copyright (C) 2024 Percona LLC
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

package tests

import (
	"bytes"
	"fmt"
	"strings"
	"testing"
	"text/tabwriter"

	"github.com/davecgh/go-spew/spew"
	"github.com/stretchr/testify/require"
	"gopkg.in/reform.v1"
)

// LogTable logs passed structures.
func LogTable(t *testing.T, structs []reform.Struct) {
	t.Helper()

	if len(structs) == 0 {
		t.Log("logTable: empty")
		return
	}

	columns := structs[0].View().Columns()
	var buf bytes.Buffer
	w := tabwriter.NewWriter(&buf, 0, 0, 1, ' ', tabwriter.Debug)
	_, err := fmt.Fprintln(w, strings.Join(columns, "\t"))
	require.NoError(t, err)
	for i, c := range columns {
		columns[i] = strings.Repeat("-", len(c))
	}
	_, err = fmt.Fprintln(w, strings.Join(columns, "\t"))
	require.NoError(t, err)

	for _, str := range structs {
		res := make([]string, len(str.Values()))
		for i, v := range str.Values() {
			res[i] = spew.Sprint(v)
		}
		fmt.Fprintf(w, "%s\n", strings.Join(res, "\t"))
	}

	require.NoError(t, w.Flush())
	t.Logf("%s:\n%s", structs[0].View().Name(), buf.Bytes())
}
