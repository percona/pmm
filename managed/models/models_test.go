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
	"testing"

	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/percona/pmm/managed/utils/tests"
)

func TestLabels(t *testing.T) {
	t.Parallel()

	t.Run("Normal", func(t *testing.T) {
		t.Parallel()

		var b []byte
		err := setLabels(map[string]string{"_1foo": "bar", "baz": "  "}, &b)
		assert.NoError(t, err)
		assert.JSONEq(t, `{"_1foo":"bar","baz":""}`, string(b))
		m, err := getLabels(b)
		assert.NoError(t, err)
		assert.Equal(t, map[string]string{"_1foo": "bar", "baz": ""}, m)
	})

	t.Run("Empty", func(t *testing.T) {
		t.Parallel()

		var b []byte
		err := setLabels(make(map[string]string), &b)
		assert.NoError(t, err)
		assert.Nil(t, b)
		m, err := getLabels(b)
		assert.NoError(t, err)
		assert.Nil(t, m)
	})

	t.Run("Invalid", func(t *testing.T) {
		t.Parallel()

		var b []byte
		err := setLabels(map[string]string{"1": "bar"}, &b)
		tests.AssertGRPCError(t, status.New(codes.InvalidArgument, `Invalid label name "1".`), err)
	})

	t.Run("Reserved", func(t *testing.T) {
		t.Parallel()

		var b []byte
		err := setLabels(map[string]string{"__1": "bar"}, &b)
		tests.AssertGRPCError(t, status.New(codes.InvalidArgument, `Invalid label name "__1".`), err)
	})
}
