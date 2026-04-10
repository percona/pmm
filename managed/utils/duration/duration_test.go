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

package duration

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/types/known/durationpb"
)

func TestOptionalFromProto(t *testing.T) {
	t.Run("nil", func(t *testing.T) {
		assert.Nil(t, OptionalFromProto(nil))
	})

	t.Run("zero", func(t *testing.T) {
		actual := OptionalFromProto(durationpb.New(0))
		if assert.NotNil(t, actual) {
			assert.Equal(t, time.Duration(0), *actual)
		}
	})

	t.Run("non-zero", func(t *testing.T) {
		actual := OptionalFromProto(durationpb.New(1500 * time.Millisecond))
		if assert.NotNil(t, actual) {
			assert.Equal(t, 1500*time.Millisecond, *actual)
		}
	})
}

func TestFromProto(t *testing.T) {
	t.Parallel()

	t.Run("nil", func(t *testing.T) {
		t.Parallel()

		assert.Zero(t, FromProto(nil))
	})

	t.Run("valid", func(t *testing.T) {
		t.Parallel()

		assert.Equal(t, 7*time.Second, FromProto(durationpb.New(7*time.Second)))
	})

	t.Run("invalid", func(t *testing.T) {
		t.Parallel()

		assert.Zero(t, FromProto(&durationpb.Duration{Seconds: 1, Nanos: -1}))
	})
}
