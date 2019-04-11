// pmm-admin
// Copyright (C) 2018 Percona LLC
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

package inventory

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseCustomLabel(t *testing.T) {
	errWrongFormat := fmt.Errorf("wrong custom label format")
	for _, v := range []struct {
		name     string
		input    string
		expected map[string]string
		expErr   error
	}{
		{"simple label", "foo=bar", map[string]string{"foo": "bar"}, nil},
		{"two labels", "foo=bar,bar=foo", map[string]string{"foo": "bar", "bar": "foo"}, nil},
		{"no value", "foo=", map[string]string(nil), errWrongFormat},
		{"no key", "=foo", map[string]string(nil), errWrongFormat},
		{"wrong format", "foo=bar,bar+foo", map[string]string(nil), errWrongFormat},
		{"empty value", "", map[string]string{}, nil},
	} {
		t.Run(v.name, func(tt *testing.T) {
			customLabels, err := parseCustomLabels(v.input)
			assert.Equal(t, v.expected, customLabels)
			assert.Equal(t, v.expErr, err)
		})
	}
}
