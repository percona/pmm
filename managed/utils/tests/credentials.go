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

package tests

import (
	"os"
	"os/user"
	"strings"
	"testing"
	"time"

	"github.com/brianvoe/gofakeit/v6"
	"github.com/stretchr/testify/require"
)

// GenEmail generates test user email.
func GenEmail(tb testing.TB) string {
	tb.Helper()
	u, err := user.Current()
	require.NoError(tb, err)

	hostname, err := os.Hostname()
	require.NoError(tb, err)

	return strings.Join([]string{u.Username, hostname, gofakeit.Email(), "test"}, ".")
}

// GenCredentials generates test user email and password.
func GenCredentials(tb testing.TB) (string, string) {
	tb.Helper()
	email := GenEmail(tb)
	password := gofakeit.Password(true, true, true, false, false, 14)
	return email, password
}

//nolint:gochecknoinits
func init() {
	gofakeit.Seed(time.Now().UnixNano())
}
