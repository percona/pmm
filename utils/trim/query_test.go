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

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestQuery(t *testing.T) {
	m := maxQueryLength
	maxQueryLength = 5
	defer func() {
		maxQueryLength = m
	}()

	assert.Equal(t, "абвг", Query("абвг"))
	assert.Equal(t, "абвгд", Query("абвгд"))
	assert.Equal(t, "а ...", Query("абвгде"))
	assert.Equal(t, "а ...", Query("абвгдеё"))

	// Unicode replacement characters
	assert.Equal(t, "\uFFFD\uFFFD\uFFFD\uFFFD\uFFFD", Query("\xff\xff\xff\xff\xff"))
	assert.Equal(t, "\uFFFD ...", Query("\xff\xff\xff\xff\xff\xff"))
}
