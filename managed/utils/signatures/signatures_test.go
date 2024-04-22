// Copyright (C) 2024 Percona LLC
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

// Package signatures verifies signatures received from Percona Platform.
package signatures

import (
	"strings"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func TestVerifySignatures(t *testing.T) {
	t.Parallel()
	l := logrus.WithField("component", "signatures-test")

	t.Run("normal", func(t *testing.T) {
		t.Parallel()

		validKey := "RWSdGihBPffV2c4IysqHAIxc5c5PLfmQStbRPkuLXDr3igJOqFWt7aml"
		invalidKey := "RWSdGihBPffV2c4IysqHAIxc5c5PLfmQStbRPkuLXDr3igJO+INVALID"

		publicKeys := []string{invalidKey, validKey}

		validSign := strings.TrimSpace(`
untrusted comment: signature from minisign secret key
RWSdGihBPffV2W/zvmIiTLh8UnocoF3OcwmczGdZ+zM13eRnm2Qq9YxfQ9cLzAp1dA5w7C5a3Cp5D7jlYiydu5hqZhJUxJt/ugg=
trusted comment: some comment
uEF33ScMPYpvHvBKv8+yBkJ9k4+DCfV4nDs6kKYwGhalvkkqwWkyfJffO+KW7a1m3y42WHpOnzBxLJeU/AuzDw==
`)

		invalidSign := strings.TrimSpace(`
untrusted comment: signature from minisign secret key
RWSdGihBPffV2W/zvmIiTLh8UnocoF3OcwmczGdZ+zM13eRnm2Qq9YxfQ9cLzAp1dA5w7C5a3Cp5D7jlYiydu5hqZhJ+INVALID=
trusted comment: some comment
uEF33ScMPYpvHvBKv8+yBkJ9k4+DCfV4nDs6kKYwGhalvkkqwWkyfJffO+KW7a1m3y42WHpOnzBxLJ+INVALID==
`)

		err := Verify(l, "random data", []string{invalidSign, validSign}, publicKeys)
		assert.NoError(t, err)
	})

	t.Run("empty signatures", func(t *testing.T) {
		t.Parallel()

		publicKeys := []string{"RWSdGihBPffV2c4IysqHAIxc5c5PLfmQStbRPkuLXDr3igJOqFWt7aml"}

		err := Verify(l, "random data", []string{}, publicKeys)
		assert.EqualError(t, err, "zero signatures received")
	})
}
