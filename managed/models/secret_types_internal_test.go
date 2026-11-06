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

package models

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

type taggedNumber struct {
	N int `encrypt:"true"`
}

type taggedString struct {
	S string `encrypt:"true"`
}

func TestApplyToSecretFieldsErrors(t *testing.T) {
	t.Parallel()

	err := applyToSecretFields(&taggedNumber{}, func(s string) (string, error) { return s, nil })
	require.ErrorContains(t, err, "field taggedNumber.N is tagged encrypt but is not a string")

	err = applyToSecretFields(&taggedString{S: "secret"}, func(string) (string, error) { return "", errors.New("cipher failed") })
	require.ErrorContains(t, err, "field taggedString.S: cipher failed")
}
