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

// Package agentlocal provides facilities for accessing local pmm-agent API.
package agentlocal

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetHostname(t *testing.T) {
	t.Parallel()

	t.Run("Returns socket", func(t *testing.T) {
		t.Parallel()
		h := GetHostname("host", 123, "/path/to/socket")
		assert.Equal(t, h, "unix-socket")
	})

	t.Run("Returns address", func(t *testing.T) {
		t.Parallel()
		h := GetHostname("host", 123, "")
		assert.Equal(t, h, "host:123")
	})
}
