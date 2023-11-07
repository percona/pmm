// Copyright (C) 2023 Percona LLC
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program. If not, see <https://www.gnu.org/licenses/>.

package convertors

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConvertToCPUMillis(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		in             string
		expectedOut    uint64
		errShouldBeNil bool
	}{
		{in: "100m", expectedOut: 100, errShouldBeNil: true},
		{in: "1", expectedOut: 1000, errShouldBeNil: true},
		{in: "1.252", expectedOut: 1252, errShouldBeNil: true},
		{in: "0.252", expectedOut: 252, errShouldBeNil: true},
		{in: "0.0", expectedOut: 0, errShouldBeNil: true},
		{in: "0.", expectedOut: 0, errShouldBeNil: true},
		{in: ".0", expectedOut: 0, errShouldBeNil: true},
		{in: ".", expectedOut: 0, errShouldBeNil: false},
		{in: "", expectedOut: 0, errShouldBeNil: true},
		{in: "adf", expectedOut: 0, errShouldBeNil: false},
	}

	for _, test := range testCases {
		out, err := StrToMilliCPU(test.in)
		assert.Equal(t, test.expectedOut, out, "in=%v, out=%v, err=%v", test.in, out, err)
		assert.Equal(
			t, test.errShouldBeNil, err == nil,
			"in=%v, out=%v, errShouldBeNil=%v: actually err == nil is %v\nerr=%v",
			test.in, out, test.errShouldBeNil, err == nil, err)
	}
}

func TestConvertToBytes(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		in             string
		expectedOut    uint64
		errShouldBeNil bool
	}{
		{in: "3000m", expectedOut: 3, errShouldBeNil: true},
		{in: "100M", expectedOut: 100 * 1000 * 1000, errShouldBeNil: true},
		{in: "100Mi", expectedOut: 100 * 1024 * 1024, errShouldBeNil: true},
		{in: "100", expectedOut: 100, errShouldBeNil: true},
		{in: "1G", expectedOut: 1000 * 1000 * 1000, errShouldBeNil: true},
		{in: "1Gi", expectedOut: 1024 * 1024 * 1024, errShouldBeNil: true},
		{in: "0.5Gi", expectedOut: 1024 * 1024 * 1024 / 2, errShouldBeNil: true},
		{in: "0.3Gi", expectedOut: 322122548, errShouldBeNil: true},
		{in: "Gi", expectedOut: 0, errShouldBeNil: false},
		{in: "", expectedOut: 0, errShouldBeNil: true},
		{in: "1Z", expectedOut: 0, errShouldBeNil: false},
		{in: "1Ki", expectedOut: 1024, errShouldBeNil: true},
		{in: "1k", expectedOut: 1000, errShouldBeNil: true},
		{in: "1T", expectedOut: 1000 * 1000 * 1000 * 1000, errShouldBeNil: true},
		{in: "1Ti", expectedOut: 1024 * 1024 * 1024 * 1024, errShouldBeNil: true},
		{in: "1P", expectedOut: 1000 * 1000 * 1000 * 1000 * 1000, errShouldBeNil: true},
		{in: "1Pi", expectedOut: 1024 * 1024 * 1024 * 1024 * 1024, errShouldBeNil: true},
		{in: "1E", expectedOut: 1000 * 1000 * 1000 * 1000 * 1000 * 1000, errShouldBeNil: true},
		{in: "1Ei", expectedOut: 1024 * 1024 * 1024 * 1024 * 1024 * 1024, errShouldBeNil: true},
	}

	for _, test := range testCases {
		out, err := StrToBytes(test.in)
		assert.Equal(t, test.expectedOut, out, "in=%v, out=%v, err=%v", test.in, out, err)
		assert.Equal(
			t, test.errShouldBeNil, err == nil,
			"in=%v, out=%v, errShouldBeNil=%v: actually err == nil is %v\nerr=%v",
			test.in, out, test.errShouldBeNil, err == nil, err)
	}
}
