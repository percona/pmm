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

package models_test

import (
	"testing"
	"time"

	"github.com/AlekSi/pointer"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/managed/utils/database"
	"github.com/percona/pmm/managed/utils/testdb"
)

func TestSettings(t *testing.T) {
	sqlDB := testdb.Open(t, database.SkipFixtures, nil)
	defer func() {
		require.NoError(t, sqlDB.Close())
	}()

	t.Run("Defaults", func(t *testing.T) {
		actual, err := models.GetSettings(sqlDB)
		require.NoError(t, err)
		require.NotEmpty(t, actual.EncryptedItems)
		expected := &models.Settings{
			MetricsResolutions: models.MetricsResolutions{
				HR: 5 * time.Second,
				MR: 10 * time.Second,
				LR: time.Minute,
			},
			DataRetention: 30 * 24 * time.Hour,
			AWSPartitions: []string{"aws"},
			SaaS: models.Advisors{
				AdvisorRunIntervals: models.AdvisorsRunIntervals{
					StandardInterval: 24 * time.Hour,
					RareInterval:     78 * time.Hour,
					FrequentInterval: 4 * time.Hour,
				},
			},
			DefaultRoleID:  1,
			EncryptedItems: actual.EncryptedItems,
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
			SaaS: models.Advisors{
				AdvisorRunIntervals: models.AdvisorsRunIntervals{
					StandardInterval: 24 * time.Hour,
					RareInterval:     78 * time.Hour,
					FrequentInterval: 4 * time.Hour,
				},
			},
		}
		assert.Equal(t, expected, s)
	})

	t.Run("Validation", func(t *testing.T) {
		t.Run("AWSPartitions", func(t *testing.T) {
			s := &models.ChangeSettingsParams{
				AWSPartitions: []string{"foo"},
			}
			_, err := models.UpdateSettings(sqlDB, s)
			var errInvalidArgument *models.InvalidArgumentError
			assert.True(t, errors.As(err, &errInvalidArgument))
			assert.EqualError(t, err, `invalid argument: aws_partitions: partition "foo" is invalid`)

			s = &models.ChangeSettingsParams{
				AWSPartitions: []string{"foo", "foo", "foo", "foo", "foo", "foo", "foo", "foo", "foo", "foo", "foo"},
			}
			_, err = models.UpdateSettings(sqlDB, s)
			assert.True(t, errors.As(err, &errInvalidArgument))
			assert.EqualError(t, err, `invalid argument: aws_partitions: list is too long`)

			s = &models.ChangeSettingsParams{
				AWSPartitions: []string{"aws", "aws-cn", "aws-cn"},
			}
			settings, err := models.UpdateSettings(sqlDB, s)
			require.NoError(t, err)
			assert.Equal(t, []string{"aws", "aws-cn"}, settings.AWSPartitions)

			// Nil is treated as not changed
			s = &models.ChangeSettingsParams{AWSPartitions: nil}
			settings, err = models.UpdateSettings(sqlDB, s)
			require.NoError(t, err)
			assert.Equal(t, []string{"aws", "aws-cn"}, settings.AWSPartitions)

			// Empty list is treated as reset to default
			s = &models.ChangeSettingsParams{AWSPartitions: []string{}}
			settings, err = models.UpdateSettings(sqlDB, s)
			require.NoError(t, err)
			assert.Equal(t, []string{"aws"}, settings.AWSPartitions)
		})

		t.Run("", func(t *testing.T) {
			mr := models.MetricsResolutions{MR: 500 * time.Millisecond} // 0.5s
			_, err := models.UpdateSettings(sqlDB, &models.ChangeSettingsParams{
				MetricsResolutions: mr,
			})
			var errInvalidArgument *models.InvalidArgumentError
			assert.True(t, errors.As(err, &errInvalidArgument))
			assert.EqualError(t, err, `invalid argument: mr: minimal resolution is 1s`)

			mr = models.MetricsResolutions{MR: 2*time.Second + (500 * time.Millisecond)} // 2.5s
			_, err = models.UpdateSettings(sqlDB, &models.ChangeSettingsParams{
				MetricsResolutions: mr,
			})
			assert.True(t, errors.As(err, &errInvalidArgument))
			assert.EqualError(t, err, `invalid argument: mr: should be a natural number of seconds`)

			_, err = models.UpdateSettings(sqlDB, &models.ChangeSettingsParams{
				DataRetention: 90000 * time.Second, // 25h
			})
			assert.True(t, errors.As(err, &errInvalidArgument))
			assert.EqualError(t, err, `invalid argument: data_retention: should be a natural number of days`)

			_, err = models.UpdateSettings(sqlDB, &models.ChangeSettingsParams{
				DataRetention: 43200 * time.Second, // 12h
			})
			assert.True(t, errors.As(err, &errInvalidArgument))
			assert.EqualError(t, err, `invalid argument: data_retention: minimal resolution is 24h`)
		})

		t.Run("Updates validation", func(t *testing.T) {
			ns, err := models.UpdateSettings(sqlDB, &models.ChangeSettingsParams{
				EnableUpdates: pointer.ToBool(true),
			})
			require.NoError(t, err)
			assert.True(t, *ns.Updates.Enabled)

			ns, err = models.UpdateSettings(sqlDB, &models.ChangeSettingsParams{
				EnableUpdates: pointer.ToBool(false),
			})
			require.NoError(t, err)
			assert.False(t, *ns.Updates.Enabled)
		})

		t.Run("Telemetry and Advisors validation", func(t *testing.T) {
			// ensure initial default state
			ns, err := models.UpdateSettings(sqlDB, &models.ChangeSettingsParams{
				EnableTelemetry: pointer.ToBool(true),
				EnableAdvisors:  pointer.ToBool(true),
			})
			require.NoError(t, err)
			assert.True(t, *ns.Telemetry.Enabled)
			assert.True(t, *ns.SaaS.Enabled)

			// disable telemetry, enable Advisors
			ns, err = models.UpdateSettings(sqlDB, &models.ChangeSettingsParams{
				EnableTelemetry: pointer.ToBool(false),
				EnableAdvisors:  pointer.ToBool(true),
			})
			require.NoError(t, err)
			assert.False(t, *ns.Telemetry.Enabled)
			assert.True(t, *ns.SaaS.Enabled)

			// disable Advisors, enable Telemetry
			ns, err = models.UpdateSettings(sqlDB, &models.ChangeSettingsParams{
				EnableTelemetry: pointer.ToBool(true),
				EnableAdvisors:  pointer.ToBool(false),
			})
			require.NoError(t, err)
			assert.True(t, *ns.Telemetry.Enabled)
			assert.False(t, *ns.SaaS.Enabled)

			// enable both
			ns, err = models.UpdateSettings(sqlDB, &models.ChangeSettingsParams{
				EnableAdvisors:  pointer.ToBool(true),
				EnableTelemetry: pointer.ToBool(true),
			})
			require.NoError(t, err)
			assert.True(t, *ns.Telemetry.Enabled)
			assert.True(t, *ns.SaaS.Enabled)

			// disable Advisors
			ns, err = models.UpdateSettings(sqlDB, &models.ChangeSettingsParams{
				EnableAdvisors: pointer.ToBool(false),
			})
			require.NoError(t, err)
			assert.True(t, *ns.Telemetry.Enabled)
			assert.False(t, *ns.SaaS.Enabled)

			// enable Advisors
			ns, err = models.UpdateSettings(sqlDB, &models.ChangeSettingsParams{
				EnableAdvisors: pointer.ToBool(true),
			})
			require.NoError(t, err)
			assert.True(t, *ns.Telemetry.Enabled)
			assert.True(t, *ns.SaaS.Enabled)

			// restore initial default state
			ns, err = models.UpdateSettings(sqlDB, &models.ChangeSettingsParams{
				EnableAdvisors:  pointer.ToBool(true),
				EnableTelemetry: pointer.ToBool(true),
			})
			require.NoError(t, err)
			assert.True(t, *ns.Telemetry.Enabled)
			assert.True(t, *ns.SaaS.Enabled)
		})

		t.Run("Check that telemetry disabling resets telemetry UUID", func(t *testing.T) {
			ns, err := models.UpdateSettings(sqlDB, &models.ChangeSettingsParams{
				EnableTelemetry: pointer.ToBool(true),
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
				EnableTelemetry: pointer.ToBool(false),
			})
			require.NoError(t, err)
			assert.Empty(t, ns.Telemetry.UUID)
		})

		t.Run("disable checks", func(t *testing.T) {
			disChecks := []string{"one", "two", "three"}

			ns, err := models.UpdateSettings(sqlDB, &models.ChangeSettingsParams{
				DisableAdvisorChecks: disChecks,
			})
			require.NoError(t, err)
			assert.ElementsMatch(t, ns.SaaS.DisabledAdvisors, disChecks)
		})

		t.Run("enable checks", func(t *testing.T) {
			disChecks := []string{"one", "two", "three"}

			_, err := models.UpdateSettings(sqlDB, &models.ChangeSettingsParams{DisableAdvisorChecks: disChecks})
			require.NoError(t, err)

			ns, err := models.UpdateSettings(sqlDB, &models.ChangeSettingsParams{EnableAdvisorChecks: []string{"two"}})
			require.NoError(t, err)
			assert.ElementsMatch(t, ns.SaaS.DisabledAdvisors, []string{"one", "three"})
		})

		t.Run("enable azure discover", func(t *testing.T) {
			s, err := models.UpdateSettings(sqlDB, &models.ChangeSettingsParams{EnableAzurediscover: pointer.ToBool(false)})
			require.NoError(t, err)
			assert.False(t, *s.Azurediscover.Enabled)

			ns, err := models.UpdateSettings(sqlDB, &models.ChangeSettingsParams{EnableAzurediscover: pointer.ToBool(true)})
			require.NoError(t, err)
			assert.True(t, *ns.Azurediscover.Enabled)
		})

		t.Run("enable Access Control", func(t *testing.T) {
			s, err := models.UpdateSettings(sqlDB, &models.ChangeSettingsParams{EnableAccessControl: pointer.ToBool(false)})
			require.NoError(t, err)
			assert.False(t, *s.AccessControl.Enabled)

			ns, err := models.UpdateSettings(sqlDB, &models.ChangeSettingsParams{EnableAccessControl: pointer.ToBool(true)})
			require.NoError(t, err)
			assert.True(t, *ns.AccessControl.Enabled)
		})

		t.Run("disable percona alerting", func(t *testing.T) {
			s, err := models.UpdateSettings(sqlDB, &models.ChangeSettingsParams{EnableAlerting: pointer.ToBool(false)})
			require.NoError(t, err)
			assert.False(t, *s.Alerting.Enabled)

			ns, err := models.UpdateSettings(sqlDB, &models.ChangeSettingsParams{EnableAlerting: pointer.ToBool(true)})
			require.NoError(t, err)
			assert.True(t, *ns.Alerting.Enabled)
		})

		t.Run("Set PMM server ID", func(t *testing.T) {
			t.Run("not set", func(t *testing.T) {
				settings, err := models.GetSettings(sqlDB)
				require.NoError(t, err)
				require.NotNil(t, settings)
				assert.Empty(t, settings.PMMServerID)

				err = models.SetPMMServerID(sqlDB)
				require.NoError(t, err)

				settings, err = models.GetSettings(sqlDB)
				require.NoError(t, err)
				require.NotNil(t, settings)
				assert.NotEmpty(t, settings.PMMServerID)
			})
			t.Run("already set", func(t *testing.T) {
				settings, err := models.GetSettings(sqlDB)
				require.NoError(t, err)
				require.NotNil(t, settings)
				pmmServerID := settings.PMMServerID

				err = models.SetPMMServerID(sqlDB)
				require.NoError(t, err)

				settings, err = models.GetSettings(sqlDB)
				require.NoError(t, err)
				require.NotNil(t, settings)
				assert.Equal(t, pmmServerID, settings.PMMServerID)
			})
		})
	})
}
