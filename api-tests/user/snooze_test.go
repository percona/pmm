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

package user

import (
	"testing"
	"time"

	"github.com/go-openapi/strfmt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	_ "github.com/percona/pmm/api-tests"
	userClient "github.com/percona/pmm/api/user/v1/json/client"
	userService "github.com/percona/pmm/api/user/v1/json/client/user_service"
)

func TestUpdateSnoozing(t *testing.T) {
	t.Run("provides default snooze information in user info", func(t *testing.T) {
		res, err1 := userClient.Default.UserService.GetUser(nil)

		require.NoError(t, err1)

		assert.Empty(t, res.Payload.SnoozedPMMVersion)
		assert.Equal(t, strfmt.DateTime(time.Time{}), res.Payload.SnoozedAt)
		assert.Equal(t, int64(0), res.Payload.SnoozeCount)
	})

	t.Run("snoozes the update", func(t *testing.T) {
		res, err1 := userClient.Default.UserService.SnoozeUpdate(&userService.SnoozeUpdateParams{
			Body: userService.SnoozeUpdateBody{
				SnoozedPMMVersion: "1.0.0",
			},
		})

		require.NoError(t, err1)

		assert.Equal(t, "1.0.0", res.Payload.SnoozedPMMVersion)
		assert.WithinDuration(t, time.Now(), time.Time(res.Payload.SnoozedAt), 1*time.Second)
		assert.Equal(t, int64(1), res.Payload.SnoozeCount)
	})

	t.Run("increments the snooze count", func(t *testing.T) {
		res, err := userClient.Default.UserService.SnoozeUpdate(&userService.SnoozeUpdateParams{
			Body: userService.SnoozeUpdateBody{
				SnoozedPMMVersion: "1.0.0",
			},
		})

		require.NoError(t, err)

		assert.Equal(t, "1.0.0", res.Payload.SnoozedPMMVersion)
		assert.WithinDuration(t, time.Now(), time.Time(res.Payload.SnoozedAt), 1*time.Second)
		assert.Equal(t, int64(2), res.Payload.SnoozeCount)
	})

	t.Run("resets the snooze count when version is different", func(t *testing.T) {
		res, err := userClient.Default.UserService.SnoozeUpdate(&userService.SnoozeUpdateParams{
			Body: userService.SnoozeUpdateBody{
				SnoozedPMMVersion: "2.0.0",
			},
		})

		require.NoError(t, err)

		assert.Equal(t, "2.0.0", res.Payload.SnoozedPMMVersion)
		assert.Equal(t, strfmt.DateTime(time.Time{}), res.Payload.SnoozedAt)
		assert.Equal(t, int64(1), res.Payload.SnoozeCount)
	})
}
