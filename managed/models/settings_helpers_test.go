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

	"github.com/brianvoe/gofakeit/v6"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/managed/utils/testdb"
	"github.com/percona/pmm/managed/utils/tests"
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
			SaaS: models.SaaS{
				STTCheckIntervals: models.STTCheckIntervals{
					StandardInterval: 24 * time.Hour,
					RareInterval:     78 * time.Hour,
					FrequentInterval: 4 * time.Hour,
				},
			},
			DefaultRoleID: 1,
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
			SaaS: models.SaaS{
				STTCheckIntervals: models.STTCheckIntervals{
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

			s = &models.ChangeSettingsParams{
				AWSPartitions: []string{},
			}
			settings, err = models.UpdateSettings(sqlDB, s)
			require.NoError(t, err)
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
			var errInvalidArgument *models.InvalidArgumentError
			assert.True(t, errors.As(err, &errInvalidArgument))
			assert.EqualError(t, err, `invalid argument: invalid alert_manager_url: mailto:hello@example.com - missing protocol scheme`)
			_, err = models.UpdateSettings(sqlDB, &models.ChangeSettingsParams{
				AlertManagerURL: "1.2.3.4:1234",
			})
			assert.True(t, errors.As(err, &errInvalidArgument))
			assert.EqualError(t, err, `invalid argument: invalid alert_manager_url: 1.2.3.4:1234 - missing protocol scheme`)
			_, err = models.UpdateSettings(sqlDB, &models.ChangeSettingsParams{
				AlertManagerURL: "1.2.3.4",
			})
			assert.True(t, errors.As(err, &errInvalidArgument))
			assert.EqualError(t, err, `invalid argument: invalid alert_manager_url: 1.2.3.4 - missing protocol scheme`)
			_, err = models.UpdateSettings(sqlDB, &models.ChangeSettingsParams{
				AlertManagerURL: "1.2.3.4//",
			})
			assert.True(t, errors.As(err, &errInvalidArgument))
			assert.EqualError(t, err, `invalid argument: invalid alert_manager_url: 1.2.3.4// - missing protocol scheme`)
			_, err = models.UpdateSettings(sqlDB, &models.ChangeSettingsParams{
				AlertManagerURL: "https://",
			})
			assert.True(t, errors.As(err, &errInvalidArgument))
			assert.EqualError(t, err, `invalid argument: invalid alert_manager_url: https:// - missing host`)
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
				DisableUpdates: false,
			})
			require.NoError(t, err)
			assert.False(t, ns.Updates.Disabled)

			_, err = models.UpdateSettings(sqlDB, &models.ChangeSettingsParams{
				EnableUpdates:  true,
				DisableUpdates: true,
			})
			var errInvalidArgument *models.InvalidArgumentError
			assert.True(t, errors.As(err, &errInvalidArgument))
			assert.EqualError(t, err, `invalid argument: both enable_updates and disable_updates are present`)

			ns, err = models.UpdateSettings(sqlDB, &models.ChangeSettingsParams{
				DisableUpdates: true,
			})
			require.NoError(t, err)
			assert.True(t, ns.Updates.Disabled)
		})

		t.Run("Telemetry and STT validation", func(t *testing.T) {
			// ensure initial default state
			ns, err := models.UpdateSettings(sqlDB, &models.ChangeSettingsParams{
				EnableTelemetry: true,
				EnableSTT:       true,
			})
			require.NoError(t, err)
			assert.False(t, ns.Telemetry.Disabled)
			assert.False(t, ns.SaaS.STTDisabled)

			_, err = models.UpdateSettings(sqlDB, &models.ChangeSettingsParams{
				EnableTelemetry:  true,
				DisableTelemetry: true,
			})
			var errInvalidArgument *models.InvalidArgumentError
			assert.True(t, errors.As(err, &errInvalidArgument))
			assert.EqualError(t, err, `invalid argument: both enable_telemetry and disable_telemetry are present`)

			_, err = models.UpdateSettings(sqlDB, &models.ChangeSettingsParams{
				EnableSTT:  true,
				DisableSTT: true,
			})
			assert.True(t, errors.As(err, &errInvalidArgument))
			assert.EqualError(t, err, `invalid argument: both enable_stt and disable_stt are present`)

			// disable telemetry, enable STT
			ns, err = models.UpdateSettings(sqlDB, &models.ChangeSettingsParams{
				DisableTelemetry: true,
				EnableSTT:        true,
			})
			require.NoError(t, err)
			assert.True(t, ns.Telemetry.Disabled)
			assert.False(t, ns.SaaS.STTDisabled)

			// disable STT, enable Telemetry
			ns, err = models.UpdateSettings(sqlDB, &models.ChangeSettingsParams{
				EnableTelemetry: true,
				DisableSTT:      true,
			})
			require.NoError(t, err)
			assert.False(t, ns.Telemetry.Disabled)
			assert.True(t, ns.SaaS.STTDisabled)

			// enable both
			ns, err = models.UpdateSettings(sqlDB, &models.ChangeSettingsParams{
				EnableSTT:       true,
				EnableTelemetry: true,
			})
			require.NoError(t, err)
			assert.False(t, ns.Telemetry.Disabled)
			assert.False(t, ns.SaaS.STTDisabled)

			// disable STT
			ns, err = models.UpdateSettings(sqlDB, &models.ChangeSettingsParams{
				DisableSTT: true,
			})
			require.NoError(t, err)
			assert.False(t, ns.Telemetry.Disabled)
			assert.True(t, ns.SaaS.STTDisabled)

			ns, err = models.UpdateSettings(sqlDB, &models.ChangeSettingsParams{
				EnableSTT: true,
			})
			require.NoError(t, err)
			assert.False(t, ns.Telemetry.Disabled)
			assert.False(t, ns.SaaS.STTDisabled)

			// restore initial default state
			ns, err = models.UpdateSettings(sqlDB, &models.ChangeSettingsParams{
				EnableTelemetry: true,
				EnableSTT:       true,
			})
			require.NoError(t, err)
			assert.False(t, ns.Telemetry.Disabled)
			assert.False(t, ns.SaaS.STTDisabled)
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

		t.Run("enable azure discover", func(t *testing.T) {
			_, err := models.UpdateSettings(sqlDB, &models.ChangeSettingsParams{DisableAzurediscover: true})
			require.NoError(t, err)

			ns, err := models.UpdateSettings(sqlDB, &models.ChangeSettingsParams{EnableAzurediscover: true})
			require.NoError(t, err)
			assert.True(t, ns.Azurediscover.Enabled)
		})

		t.Run("enable Access Control", func(t *testing.T) {
			s, err := models.UpdateSettings(sqlDB, &models.ChangeSettingsParams{DisableAccessControl: true})
			require.NoError(t, err)
			assert.False(t, s.AccessControl.Enabled)

			ns, err := models.UpdateSettings(sqlDB, &models.ChangeSettingsParams{EnableAccessControl: true})
			require.NoError(t, err)
			assert.True(t, ns.AccessControl.Enabled)
		})

		t.Run("Integrated Alerting settings validation", func(t *testing.T) {
			emailSettings := &models.EmailAlertingSettings{
				From:      tests.GenEmail(t),
				Smarthost: "0.0.0.0:8080",
				Hello:     "smtp_host",
				Username:  "smtp_username",
				Password:  "smtp_password",
				Secret:    "smtp_secret",
			}
			slackSettings := &models.SlackAlertingSettings{URL: gofakeit.URL()}
			ns, err := models.UpdateSettings(sqlDB, &models.ChangeSettingsParams{
				EnableAlerting:        true,
				EmailAlertingSettings: emailSettings,
				SlackAlertingSettings: slackSettings,
			})
			require.NoError(t, err)
			assert.False(t, ns.Alerting.Disabled)
			assert.Equal(t, ns.Alerting.EmailAlertingSettings, emailSettings)
			assert.Equal(t, ns.Alerting.SlackAlertingSettings, slackSettings)

			// check that we don't lose settings on empty updates
			ns, err = models.UpdateSettings(sqlDB, &models.ChangeSettingsParams{})
			require.NoError(t, err)
			assert.False(t, ns.Alerting.Disabled)
			assert.Equal(t, ns.Alerting.EmailAlertingSettings, emailSettings)
			assert.Equal(t, ns.Alerting.SlackAlertingSettings, slackSettings)

			_, err = models.UpdateSettings(sqlDB, &models.ChangeSettingsParams{
				RemoveEmailAlertingSettings: true,
				EmailAlertingSettings:       emailSettings,
			})
			var errInvalidArgument *models.InvalidArgumentError
			assert.True(t, errors.As(err, &errInvalidArgument))
			assert.EqualError(t, err, "invalid argument: both email_alerting_settings and remove_email_alerting_settings are present")
			_, err = models.UpdateSettings(sqlDB, &models.ChangeSettingsParams{
				EmailAlertingSettings: &models.EmailAlertingSettings{
					From:      "from",
					Smarthost: "example.com:1234",
					Hello:     "example.com",
				},
			})
			assert.True(t, errors.As(err, &errInvalidArgument))
			assert.EqualError(t, err, "invalid argument: invalid \"from\" email \"from\"")

			_, err = models.UpdateSettings(sqlDB, &models.ChangeSettingsParams{
				EmailAlertingSettings: &models.EmailAlertingSettings{
					From:      "from@example.com",
					Smarthost: "@invalid-host",
					Hello:     "example.com",
				},
			})
			assert.True(t, errors.As(err, &errInvalidArgument))
			assert.EqualError(t, err, "invalid argument: invalid server address, expected format host:port")

			_, err = models.UpdateSettings(sqlDB, &models.ChangeSettingsParams{
				EmailAlertingSettings: &models.EmailAlertingSettings{
					From:      "from@example.com",
					Smarthost: "example.com:1234",
					Hello:     "%",
				},
			})
			assert.True(t, errors.As(err, &errInvalidArgument))
			assert.EqualError(t, err, "invalid argument: invalid hello field, expected valid host")

			_, err = models.UpdateSettings(sqlDB, &models.ChangeSettingsParams{
				RemoveSlackAlertingSettings: true,
				SlackAlertingSettings:       slackSettings,
			})
			assert.True(t, errors.As(err, &errInvalidArgument))
			assert.EqualError(t, err, "invalid argument: both slack_alerting_settings and remove_slack_alerting_settings are present")

			_, err = models.UpdateSettings(sqlDB, &models.ChangeSettingsParams{
				SlackAlertingSettings: &models.SlackAlertingSettings{
					URL: "invalid@url",
				},
			})
			assert.True(t, errors.As(err, &errInvalidArgument))
			assert.EqualError(t, err, "invalid argument: invalid url value")

			ns, err = models.UpdateSettings(sqlDB, &models.ChangeSettingsParams{
				DisableAlerting:             true,
				RemoveEmailAlertingSettings: true,
				RemoveSlackAlertingSettings: true,
			})
			require.NoError(t, err)
			assert.Empty(t, ns.Alerting.EmailAlertingSettings)
			assert.True(t, ns.Alerting.Disabled)

			_, err = models.UpdateSettings(sqlDB, &models.ChangeSettingsParams{
				DisableAlerting: true,
				EnableAlerting:  true,
			})
			assert.True(t, errors.As(err, &errInvalidArgument))
			assert.EqualError(t, err, "invalid argument: both enable_alerting and disable_alerting are present")
		})
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
}
