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
	"testing"
)

// GetAWSKeys returns testing AWS keys.
func GetAWSKeys(tb testing.TB) (string, string) {
	tb.Helper()

	accessKey, secretKey := os.Getenv("AWS_ACCESS_KEY"), os.Getenv("AWS_SECRET_KEY")
	if accessKey == "" || secretKey == "" {
		tb.Skip("Environment variables AWS_ACCESS_KEY / AWS_SECRET_KEY are not defined, skipping test")
	}
	return accessKey, secretKey
}
