// pmm-admin
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

package commands

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
		{"no value", "foo=", nil, errWrongFormat},
		{"no key", "=foo", nil, errWrongFormat},
		{"wrong format", "foo=bar,bar+foo", nil, errWrongFormat},
		{"empty value", "", map[string]string{}, nil},
		{"PMM-4078 hyphen", "region=us-east1, mylabel=mylab-22", map[string]string{"region": "us-east1", "mylabel": "mylab-22"}, nil},
	} {
		t.Run(v.name, func(t *testing.T) {
			customLabels, err := ParseCustomLabels(v.input)
			assert.Equal(t, v.expected, customLabels)
			assert.Equal(t, v.expErr, err)
		})
	}
}
