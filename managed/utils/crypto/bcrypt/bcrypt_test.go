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

package bcrypt_test

import (
	"testing"

	"github.com/percona/pmm/managed/utils/crypto/bcrypt"
	"github.com/stretchr/testify/require"
)

// Verify that the generated hashes match the expected values.
// Passwords, salts and expected hashes come from the original implementation (our fork of crypto)
// and are preserved here for compatibility.
func TestGenerateFromPasswordAndSalt(t *testing.T) {
	passwords := []string{"password", "password2"}
	salts := []string{"salt", "salt2"}
	hashed := []string{
		"$2a$10$a0Dqb.\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00IpP9MckeNpzVzfsitFCjAfJfE9QJxG6",
		"$2a$10$a0DqbBG\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00gb.ywRiNhHOv75ZkOoVpLtz9HWNLEXu",
	}

	for i := range passwords {
		buf, err := bcrypt.GenerateFromPasswordAndSalt([]byte(passwords[i]), bcrypt.DefaultCost, []byte(salts[i]))
		require.NoError(t, err)
		require.Equal(t, hashed[i], string(buf), "unexpected hash output")
	}
}
