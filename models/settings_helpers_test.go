// pmm-managed
// Copyright (C) 2017 Percona LLC
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

package models_test

import (
	"testing"
	"time"

	"github.com/brianvoe/gofakeit"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/percona/pmm-managed/models"
	"github.com/percona/pmm-managed/utils/testdb"
)

func TestSettings(t *testing.T) {
	sqlDB := testdb.Open(t, models.SkipFixtures, nil)
	defer func() {
		require.NoError(t, sqlDB.Close())
	}()

	t.Run("Defaults", func(t *testing.T) {
		actual, err := models.GetSettings(sqlDB)
		require.NoError(t, err)
		expected := &models.Settings{
			MetricsResolutions: models.MetricsResolutions{
				HR: 5 * time.Second,
				MR: 10 * time.Second,
				LR: time.Minute,
			},
			DataRetention: 30 * 24 * time.Hour,
			AWSPartitions: []string{"aws"},
		}
		assert.Equal(t, expected, actual)
	})

	t.Run("SaveWithDefaults", func(t *testing.T) {
		s := &models.Settings{}
		err := models.SaveSettings(sqlDB, s)
		require.NoError(t, err)
		expected := &models.Settings{
			MetricsResolutions: models.MetricsResolutions{
				HR: 5 * time.Second,
				MR: 10 * time.Second,
				LR: time.Minute,
			},
			DataRetention: 30 * 24 * time.Hour,
			AWSPartitions: []string{"aws"},
		}
		assert.Equal(t, expected, s)
	})

	t.Run("Validation", func(t *testing.T) {
		t.Run("AWSPartitions", func(t *testing.T) {
			s := &models.ChangeSettingsParams{
				AWSPartitions: []string{"foo"},
			}
			_, err := models.UpdateSettings(sqlDB, s)
			assert.EqualError(t, err, `aws_partitions: partition "foo" is invalid`)

			s = &models.ChangeSettingsParams{
				AWSPartitions: []string{"foo", "foo", "foo", "foo", "foo", "foo", "foo", "foo", "foo", "foo", "foo"},
			}
			_, err = models.UpdateSettings(sqlDB, s)
			assert.EqualError(t, err, `aws_partitions: list is too long`)

			s = &models.ChangeSettingsParams{
				AWSPartitions: []string{"aws", "aws-cn", "aws-cn"},
			}
			settings, err := models.UpdateSettings(sqlDB, s)
			assert.NoError(t, err)
			assert.Equal(t, []string{"aws", "aws-cn"}, settings.AWSPartitions)

			s = &models.ChangeSettingsParams{
				AWSPartitions: []string{},
			}
			settings, err = models.UpdateSettings(sqlDB, s)
			assert.NoError(t, err)
			assert.Equal(t, []string{"aws", "aws-cn"}, settings.AWSPartitions)

			settings = &models.Settings{AWSPartitions: []string{}}
			err = models.SaveSettings(sqlDB, settings)
			assert.NoError(t, err)
			assert.Equal(t, []string{"aws"}, settings.AWSPartitions)
		})

		t.Run("AlertManagerURL", func(t *testing.T) {
			_, err := models.UpdateSettings(sqlDB, &models.ChangeSettingsParams{
				AlertManagerURL: "mailto:hello@example.com",
			})
			assert.EqualError(t, err, `Invalid alert_manager_url: mailto:hello@example.com - missing protocol scheme.`)
			_, err = models.UpdateSettings(sqlDB, &models.ChangeSettingsParams{
				AlertManagerURL: "1.2.3.4:1234",
			})
			assert.EqualError(t, err, `Invalid alert_manager_url: 1.2.3.4:1234 - missing protocol scheme.`)
			_, err = models.UpdateSettings(sqlDB, &models.ChangeSettingsParams{
				AlertManagerURL: "1.2.3.4",
			})
			assert.EqualError(t, err, `Invalid alert_manager_url: 1.2.3.4 - missing protocol scheme.`)
			_, err = models.UpdateSettings(sqlDB, &models.ChangeSettingsParams{
				AlertManagerURL: "1.2.3.4//",
			})
			assert.EqualError(t, err, `Invalid alert_manager_url: 1.2.3.4// - missing protocol scheme.`)
			_, err = models.UpdateSettings(sqlDB, &models.ChangeSettingsParams{
				AlertManagerURL: "https://",
			})
			assert.EqualError(t, err, `Invalid alert_manager_url: https:// - missing host.`)
			_, err = models.UpdateSettings(sqlDB, &models.ChangeSettingsParams{
				AlertManagerURL: "https://1.2.3.4",
			})
			assert.NoError(t, err)
			_, err = models.UpdateSettings(sqlDB, &models.ChangeSettingsParams{
				AlertManagerURL: "https://1.2.3.4:1234/",
			})
			assert.NoError(t, err)
		})

		t.Run("", func(t *testing.T) {
			mr := models.MetricsResolutions{MR: 5e+8 * time.Nanosecond} // 0.5s
			_, err := models.UpdateSettings(sqlDB, &models.ChangeSettingsParams{
				MetricsResolutions: mr,
			})
			assert.EqualError(t, err, `mr: minimal resolution is 1s`)

			mr = models.MetricsResolutions{MR: 2*time.Second + 5e8*time.Nanosecond} // 2.5s
			_, err = models.UpdateSettings(sqlDB, &models.ChangeSettingsParams{
				MetricsResolutions: mr,
			})
			assert.EqualError(t, err, `mr: should be a natural number of seconds`)

			_, err = models.UpdateSettings(sqlDB, &models.ChangeSettingsParams{
				DataRetention: 90000 * time.Second, // 25h
			})
			assert.EqualError(t, err, `data_retention: should be a natural number of days`)

			_, err = models.UpdateSettings(sqlDB, &models.ChangeSettingsParams{
				DataRetention: 43200 * time.Second, // 12h
			})
			assert.EqualError(t, err, `data_retention: minimal resolution is 24h`)
		})

		t.Run("Telemetry and STT validation", func(t *testing.T) {
			// ensure initial default state
			ns, err := models.UpdateSettings(sqlDB, &models.ChangeSettingsParams{
				EnableTelemetry: true,
				DisableSTT:      true,
			})
			assert.NoError(t, err)
			assert.False(t, ns.Telemetry.Disabled)
			assert.False(t, ns.SaaS.STTEnabled)

			_, err = models.UpdateSettings(sqlDB, &models.ChangeSettingsParams{
				EnableTelemetry:  true,
				DisableTelemetry: true,
			})
			assert.EqualError(t, err, `Both enable_telemetry and disable_telemetry are present.`)

			_, err = models.UpdateSettings(sqlDB, &models.ChangeSettingsParams{
				EnableSTT:  true,
				DisableSTT: true,
			})
			assert.EqualError(t, err, `Both enable_stt and disable_stt are present.`)

			_, err = models.UpdateSettings(sqlDB, &models.ChangeSettingsParams{
				EnableSTT:        true,
				DisableTelemetry: true,
			})
			assert.EqualError(t, err, `Cannot enable STT while disabling telemetry.`)

			// enable both
			ns, err = models.UpdateSettings(sqlDB, &models.ChangeSettingsParams{
				EnableSTT:       true,
				EnableTelemetry: true,
			})
			assert.NoError(t, err)
			assert.False(t, ns.Telemetry.Disabled)
			assert.True(t, ns.SaaS.STTEnabled)

			_, err = models.UpdateSettings(sqlDB, &models.ChangeSettingsParams{
				DisableTelemetry: true,
			})
			assert.EqualError(t, err, `Cannot disable telemetry while STT is enabled.`)

			// disable both
			ns, err = models.UpdateSettings(sqlDB, &models.ChangeSettingsParams{
				DisableSTT:       true,
				DisableTelemetry: true,
			})
			assert.NoError(t, err)
			assert.True(t, ns.Telemetry.Disabled)
			assert.False(t, ns.SaaS.STTEnabled)

			_, err = models.UpdateSettings(sqlDB, &models.ChangeSettingsParams{
				EnableSTT: true,
			})
			assert.EqualError(t, err, `Cannot enable STT while telemetry is disabled.`)

			// restore initial default state
			ns, err = models.UpdateSettings(sqlDB, &models.ChangeSettingsParams{
				EnableTelemetry: true,
				DisableSTT:      true,
			})
			assert.NoError(t, err)
			assert.False(t, ns.Telemetry.Disabled)
			assert.False(t, ns.SaaS.STTEnabled)
		})

		t.Run("Check that telemetry disabling resets telemetry UUID", func(t *testing.T) {
			ns, err := models.UpdateSettings(sqlDB, &models.ChangeSettingsParams{
				EnableTelemetry: true,
			})
			require.NoError(t, err)

			uuid := "d4331513e0574eab9fe47cbd8a5f2110"
			ns.Telemetry.UUID = uuid
			err = models.SaveSettings(sqlDB, ns)
			require.NoError(t, err)

			ns, err = models.GetSettings(sqlDB)
			require.NoError(t, err)
			assert.Equal(t, uuid, ns.Telemetry.UUID)

			ns, err = models.UpdateSettings(sqlDB, &models.ChangeSettingsParams{
				DisableTelemetry: true,
			})
			require.NoError(t, err)
			assert.Empty(t, ns.Telemetry.UUID)
		})

		t.Run("Percona Platform auth ", func(t *testing.T) {
			email := gofakeit.Email()
			sessionID := gofakeit.UUID()

			// User logged in
			ns, err := models.UpdateSettings(sqlDB, &models.ChangeSettingsParams{
				Email:     email,
				SessionID: sessionID,
			})
			require.NoError(t, err)
			assert.Equal(t, email, ns.SaaS.Email)
			assert.Equal(t, sessionID, ns.SaaS.SessionID)

			// Logout with email update
			_, err = models.UpdateSettings(sqlDB, &models.ChangeSettingsParams{
				LogOut: true,
				Email:  gofakeit.Email(),
			})
			assert.Error(t, err, "Cannot logout while updating Percona Platform user data.")

			// Logout with session ID update
			_, err = models.UpdateSettings(sqlDB, &models.ChangeSettingsParams{
				LogOut:    true,
				SessionID: gofakeit.UUID(),
			})
			assert.Error(t, err, "Cannot logout while updating Percona Platform user data.")

			// Logout with email and session ID update
			_, err = models.UpdateSettings(sqlDB, &models.ChangeSettingsParams{
				LogOut:    true,
				Email:     gofakeit.Email(),
				SessionID: gofakeit.UUID(),
			})
			assert.Error(t, err, "Cannot logout while updating Percona Platform user data.")

			// Normal logout
			ns, err = models.UpdateSettings(sqlDB, &models.ChangeSettingsParams{
				LogOut: true,
			})
			require.NoError(t, err)
			assert.Empty(t, ns.SaaS.Email)
			assert.Empty(t, ns.SaaS.SessionID)
		})

		t.Run("disable checks", func(t *testing.T) {
			disChecks := []string{"one", "two", "three"}

			ns, err := models.UpdateSettings(sqlDB, &models.ChangeSettingsParams{
				DisableSTTChecks: disChecks,
			})
			require.NoError(t, err)
			assert.ElementsMatch(t, ns.SaaS.DisabledSTTChecks, disChecks)
		})

		t.Run("enable checks", func(t *testing.T) {
			disChecks := []string{"one", "two", "three"}

			_, err := models.UpdateSettings(sqlDB, &models.ChangeSettingsParams{DisableSTTChecks: disChecks})
			require.NoError(t, err)

			ns, err := models.UpdateSettings(sqlDB, &models.ChangeSettingsParams{EnableSTTChecks: []string{"two"}})
			require.NoError(t, err)
			assert.ElementsMatch(t, ns.SaaS.DisabledSTTChecks, []string{"one", "three"})
		})
	})
}
