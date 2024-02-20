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

package server

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/brianvoe/gofakeit/v6"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"

	pmmapitests "github.com/percona/pmm/api-tests"
	"github.com/percona/pmm/api/alertmanager/amclient"
	"github.com/percona/pmm/api/alertmanager/amclient/alert"
	serverClient "github.com/percona/pmm/api/serverpb/json/client"
	"github.com/percona/pmm/api/serverpb/json/client/server"
)

func TestSettings(t *testing.T) {
	t.Run("GetSettings", func(t *testing.T) {
		res, err := serverClient.Default.Server.GetSettings(nil)
		require.NoError(t, err)
		assert.True(t, res.Payload.Settings.TelemetryEnabled)
		assert.True(t, res.Payload.Settings.SttEnabled)
		expected := &server.GetSettingsOKBodySettingsMetricsResolutions{
			Hr: "5s",
			Mr: "10s",
			Lr: "60s",
		}
		assert.Equal(t, expected, res.Payload.Settings.MetricsResolutions)
		expectedSTTCheckIntervals := &server.GetSettingsOKBodySettingsSttCheckIntervals{
			FrequentInterval: "14400s",
			StandardInterval: "86400s",
			RareInterval:     "280800s",
		}
		assert.Equal(t, expectedSTTCheckIntervals, res.Payload.Settings.SttCheckIntervals)
		assert.Equal(t, "2592000s", res.Payload.Settings.DataRetention)
		assert.Equal(t, []string{"aws"}, res.Payload.Settings.AWSPartitions)
		assert.False(t, res.Payload.Settings.UpdatesDisabled)
		assert.True(t, res.Payload.Settings.AlertingEnabled)
		assert.Empty(t, res.Payload.Settings.EmailAlertingSettings)
		assert.Empty(t, res.Payload.Settings.SlackAlertingSettings)

		t.Run("ChangeSettings", func(t *testing.T) {
			defer restoreSettingsDefaults(t)

			t.Run("Updates", func(t *testing.T) {
				t.Run("DisableAndEnableUpdatesSettingsUpdate", func(t *testing.T) {
					defer restoreSettingsDefaults(t)
					res, err := serverClient.Default.Server.ChangeSettings(&server.ChangeSettingsParams{
						Body: server.ChangeSettingsBody{
							DisableUpdates: true,
						},
						Context: pmmapitests.Context,
					})
					require.NoError(t, err)
					assert.True(t, res.Payload.Settings.UpdatesDisabled)

					resg, err := serverClient.Default.Server.GetSettings(nil)
					require.NoError(t, err)
					assert.True(t, resg.Payload.Settings.UpdatesDisabled)

					res, err = serverClient.Default.Server.ChangeSettings(&server.ChangeSettingsParams{
						Body: server.ChangeSettingsBody{
							EnableUpdates: true,
						},
						Context: pmmapitests.Context,
					})
					require.NoError(t, err)
					assert.False(t, res.Payload.Settings.UpdatesDisabled)

					resg, err = serverClient.Default.Server.GetSettings(nil)
					require.NoError(t, err)
					assert.False(t, resg.Payload.Settings.UpdatesDisabled)
				})

				t.Run("InvalidBothEnableAndDisableUpdates", func(t *testing.T) {
					defer restoreSettingsDefaults(t)

					res, err := serverClient.Default.Server.ChangeSettings(&server.ChangeSettingsParams{
						Body: server.ChangeSettingsBody{
							EnableUpdates:  true,
							DisableUpdates: true,
						},
						Context: pmmapitests.Context,
					})
					pmmapitests.AssertAPIErrorf(t, err, 400, codes.InvalidArgument,
						`Invalid argument: both enable_updates and disable_updates are present.`)
					assert.Empty(t, res)
				})
			})

			t.Run("ValidAlertingSettingsUpdate", func(t *testing.T) {
				defer restoreSettingsDefaults(t)

				email := gofakeit.Email()
				smarthost := "0.0.0.0:8080"
				username := "username"
				password := "password"
				identity := "identity"
				secret := "secret"
				slackURL := gofakeit.URL()
				res, err := serverClient.Default.Server.ChangeSettings(&server.ChangeSettingsParams{
					Body: server.ChangeSettingsBody{
						EnableAlerting: true,
						EmailAlertingSettings: &server.ChangeSettingsParamsBodyEmailAlertingSettings{
							From:      email,
							Smarthost: smarthost,
							Username:  username,
							Password:  password,
							Identity:  identity,
							Secret:    secret,
						},
						SlackAlertingSettings: &server.ChangeSettingsParamsBodySlackAlertingSettings{
							URL: slackURL,
						},
					},
					Context: pmmapitests.Context,
				})
				require.NoError(t, err)
				assert.True(t, res.Payload.Settings.AlertingEnabled)
				assert.Equal(t, email, res.Payload.Settings.EmailAlertingSettings.From)
				assert.Equal(t, smarthost, res.Payload.Settings.EmailAlertingSettings.Smarthost)
				assert.Equal(t, username, res.Payload.Settings.EmailAlertingSettings.Username)
				// check that we don't expose password through the API.
				assert.Empty(t, res.Payload.Settings.EmailAlertingSettings.Password)
				assert.Equal(t, identity, res.Payload.Settings.EmailAlertingSettings.Identity)
				assert.Equal(t, secret, res.Payload.Settings.EmailAlertingSettings.Secret)
				assert.Equal(t, slackURL, res.Payload.Settings.SlackAlertingSettings.URL)
			})

			t.Run("InvalidBothSlackAlertingSettingsAndRemoveSlackAlertingSettings", func(t *testing.T) {
				defer restoreSettingsDefaults(t)

				res, err := serverClient.Default.Server.ChangeSettings(&server.ChangeSettingsParams{
					Body: server.ChangeSettingsBody{
						SlackAlertingSettings: &server.ChangeSettingsParamsBodySlackAlertingSettings{
							URL: gofakeit.URL(),
						},
						RemoveSlackAlertingSettings: true,
					},
					Context: pmmapitests.Context,
				})
				pmmapitests.AssertAPIErrorf(t, err, 400, codes.InvalidArgument,
					`Invalid argument: both slack_alerting_settings and remove_slack_alerting_settings are present.`)
				assert.Empty(t, res)
			})

			t.Run("InvalidBothEmailAlertingSettingsAndRemoveEmailAlertingSettings", func(t *testing.T) {
				defer restoreSettingsDefaults(t)

				res, err := serverClient.Default.Server.ChangeSettings(&server.ChangeSettingsParams{
					Body: server.ChangeSettingsBody{
						EmailAlertingSettings: &server.ChangeSettingsParamsBodyEmailAlertingSettings{
							From:      gofakeit.Email(),
							Smarthost: "0.0.0.0:8080",
							Username:  "username",
							Password:  "password",
							Identity:  "identity",
							Secret:    "secret",
						},
						RemoveEmailAlertingSettings: true,
					},
					Context: pmmapitests.Context,
				})
				pmmapitests.AssertAPIErrorf(t, err, 400, codes.InvalidArgument,
					`Invalid argument: both email_alerting_settings and remove_email_alerting_settings are present.`)
				assert.Empty(t, res)
			})

			t.Run("InvalidBothEnableAndDisableSTT", func(t *testing.T) {
				defer restoreSettingsDefaults(t)

				res, err := serverClient.Default.Server.ChangeSettings(&server.ChangeSettingsParams{
					Body: server.ChangeSettingsBody{
						EnableStt:  true,
						DisableStt: true,
					},
					Context: pmmapitests.Context,
				})
				pmmapitests.AssertAPIErrorf(t, err, 400, codes.InvalidArgument,
					`Invalid argument: both enable_stt and disable_stt are present.`)
				assert.Empty(t, res)
			})

			t.Run("EnableSTTAndEnableTelemetry", func(t *testing.T) {
				defer restoreSettingsDefaults(t)

				res, err := serverClient.Default.Server.ChangeSettings(&server.ChangeSettingsParams{
					Body: server.ChangeSettingsBody{
						EnableStt:       true,
						EnableTelemetry: true,
					},
					Context: pmmapitests.Context,
				})
				require.NoError(t, err)
				assert.True(t, res.Payload.Settings.SttEnabled)
				assert.True(t, res.Payload.Settings.TelemetryEnabled)

				resg, err := serverClient.Default.Server.GetSettings(nil)
				require.NoError(t, err)
				assert.True(t, resg.Payload.Settings.TelemetryEnabled)
				assert.True(t, resg.Payload.Settings.SttEnabled)
			})

			t.Run("EnableSTTAndDisableTelemetry", func(t *testing.T) {
				defer restoreSettingsDefaults(t)

				res, err := serverClient.Default.Server.ChangeSettings(&server.ChangeSettingsParams{
					Body: server.ChangeSettingsBody{
						EnableStt:        true,
						DisableTelemetry: true,
					},
					Context: pmmapitests.Context,
				})
				require.NoError(t, err)
				assert.True(t, res.Payload.Settings.SttEnabled)
				assert.False(t, res.Payload.Settings.TelemetryEnabled)
			})

			t.Run("DisableSTTAndEnableTelemetry", func(t *testing.T) {
				defer restoreSettingsDefaults(t)

				res, err := serverClient.Default.Server.ChangeSettings(&server.ChangeSettingsParams{
					Body: server.ChangeSettingsBody{
						DisableStt:      true,
						EnableTelemetry: true,
					},
					Context: pmmapitests.Context,
				})
				require.NoError(t, err)
				assert.False(t, res.Payload.Settings.SttEnabled)
				assert.True(t, res.Payload.Settings.TelemetryEnabled)

				resg, err := serverClient.Default.Server.GetSettings(nil)
				require.NoError(t, err)
				assert.True(t, resg.Payload.Settings.TelemetryEnabled)
				assert.False(t, resg.Payload.Settings.SttEnabled)
			})

			t.Run("DisableSTTAndDisableTelemetry", func(t *testing.T) {
				defer restoreSettingsDefaults(t)

				res, err := serverClient.Default.Server.ChangeSettings(&server.ChangeSettingsParams{
					Body: server.ChangeSettingsBody{
						DisableStt:       true,
						DisableTelemetry: true,
					},
					Context: pmmapitests.Context,
				})
				require.NoError(t, err)
				assert.False(t, res.Payload.Settings.SttEnabled)
				assert.False(t, res.Payload.Settings.TelemetryEnabled)

				resg, err := serverClient.Default.Server.GetSettings(nil)
				require.NoError(t, err)
				assert.False(t, resg.Payload.Settings.TelemetryEnabled)
				assert.False(t, resg.Payload.Settings.SttEnabled)
			})

			t.Run("EnableSTTWhileTelemetryEnabled", func(t *testing.T) {
				defer restoreSettingsDefaults(t)

				// Ensure Telemetry is enabled
				res, err := serverClient.Default.Server.ChangeSettings(&server.ChangeSettingsParams{
					Body: server.ChangeSettingsBody{
						EnableTelemetry: true,
					},
					Context: pmmapitests.Context,
				})
				require.NoError(t, err)
				assert.True(t, res.Payload.Settings.TelemetryEnabled)

				res, err = serverClient.Default.Server.ChangeSettings(&server.ChangeSettingsParams{
					Body: server.ChangeSettingsBody{
						EnableStt: true,
					},
					Context: pmmapitests.Context,
				})
				require.NoError(t, err)
				assert.True(t, res.Payload.Settings.SttEnabled)

				resg, err := serverClient.Default.Server.GetSettings(nil)
				require.NoError(t, err)
				assert.True(t, resg.Payload.Settings.TelemetryEnabled)
				assert.True(t, resg.Payload.Settings.SttEnabled)
			})

			t.Run("VerifyFailedChecksInAlertmanager", func(t *testing.T) {
				if !pmmapitests.RunSTTTests {
					t.Skip("Skipping STT tests until we have environment: https://jira.percona.com/browse/PMM-5106")
				}

				defer restoreSettingsDefaults(t)

				// Enabling STT
				res, err := serverClient.Default.Server.ChangeSettings(&server.ChangeSettingsParams{
					Body: server.ChangeSettingsBody{
						EnableStt: true,
					},
					Context: pmmapitests.Context,
				})
				require.NoError(t, err)
				assert.True(t, res.Payload.Settings.TelemetryEnabled)

				// 120 sec ping for failed checks alerts to appear in alertmanager
				var alertsCount int
				for i := 0; i < 120; i++ {
					res, err := amclient.Default.Alert.GetAlerts(&alert.GetAlertsParams{
						Filter:  []string{"stt_check=1"},
						Context: pmmapitests.Context,
					})
					require.NoError(t, err)
					if len(res.Payload) == 0 {
						time.Sleep(1 * time.Second)
						continue
					}

					for _, v := range res.Payload {
						t.Logf("%+v", v)

						assert.Contains(t, v.Annotations, "summary")

						assert.Equal(t, "1", v.Labels["stt_check"])

						assert.Contains(t, v.Labels, "agent_id")
						assert.Contains(t, v.Labels, "agent_type")
						assert.Contains(t, v.Labels, "alert_id")
						assert.Contains(t, v.Labels, "alertname")
						assert.Contains(t, v.Labels, "node_id")
						assert.Contains(t, v.Labels, "node_name")
						assert.Contains(t, v.Labels, "node_type")
						assert.Contains(t, v.Labels, "service_id")
						assert.Contains(t, v.Labels, "service_name")
						assert.Contains(t, v.Labels, "service_type")
						assert.Contains(t, v.Labels, "severity")
					}
					alertsCount = len(res.Payload)
					break
				}
				assert.Greater(t, alertsCount, 0, "No alerts met")
			})

			t.Run("DisableSTTWhileItIsDisabled", func(t *testing.T) {
				defer restoreSettingsDefaults(t)

				res, err := serverClient.Default.Server.ChangeSettings(&server.ChangeSettingsParams{
					Body: server.ChangeSettingsBody{
						DisableStt: true,
					},
					Context: pmmapitests.Context,
				})
				require.NoError(t, err)
				assert.False(t, res.Payload.Settings.SttEnabled)

				resg, err := serverClient.Default.Server.GetSettings(nil)
				require.NoError(t, err)
				assert.True(t, resg.Payload.Settings.TelemetryEnabled)
				assert.False(t, resg.Payload.Settings.SttEnabled)
			})

			t.Run("STTEnabledState", func(t *testing.T) {
				defer restoreSettingsDefaults(t)

				res, err := serverClient.Default.Server.ChangeSettings(&server.ChangeSettingsParams{
					Body: server.ChangeSettingsBody{
						EnableStt: true,
					},
					Context: pmmapitests.Context,
				})

				require.NoError(t, err)
				assert.True(t, res.Payload.Settings.SttEnabled)

				resg, err := serverClient.Default.Server.GetSettings(nil)
				require.NoError(t, err)
				assert.True(t, resg.Payload.Settings.TelemetryEnabled)
				assert.True(t, resg.Payload.Settings.SttEnabled)

				t.Run("EnableSTTWhileItIsEnabled", func(t *testing.T) {
					res, err := serverClient.Default.Server.ChangeSettings(&server.ChangeSettingsParams{
						Body: server.ChangeSettingsBody{
							EnableStt: true,
						},
						Context: pmmapitests.Context,
					})
					require.NoError(t, err)
					assert.True(t, res.Payload.Settings.SttEnabled)

					resg, err := serverClient.Default.Server.GetSettings(nil)
					require.NoError(t, err)
					assert.True(t, resg.Payload.Settings.TelemetryEnabled)
					assert.True(t, resg.Payload.Settings.SttEnabled)
				})

				t.Run("DisableTelemetryWhileSTTEnabled", func(t *testing.T) {
					res, err := serverClient.Default.Server.ChangeSettings(&server.ChangeSettingsParams{
						Body: server.ChangeSettingsBody{
							DisableTelemetry: true,
						},
						Context: pmmapitests.Context,
					})
					require.NoError(t, err)
					assert.True(t, res.Payload.Settings.SttEnabled)
					assert.False(t, res.Payload.Settings.TelemetryEnabled)
				})
			})

			t.Run("TelemetryDisabledState", func(t *testing.T) {
				defer restoreSettingsDefaults(t)

				res, err := serverClient.Default.Server.ChangeSettings(&server.ChangeSettingsParams{
					Body: server.ChangeSettingsBody{
						DisableTelemetry: true,
					},
					Context: pmmapitests.Context,
				})

				require.NoError(t, err)
				assert.False(t, res.Payload.Settings.TelemetryEnabled)

				resg, err := serverClient.Default.Server.GetSettings(nil)
				require.NoError(t, err)
				assert.False(t, resg.Payload.Settings.TelemetryEnabled)
				assert.True(t, resg.Payload.Settings.SttEnabled)

				t.Run("EnableSTTWhileTelemetryDisabled", func(t *testing.T) {
					defer restoreSettingsDefaults(t)

					res, err := serverClient.Default.Server.ChangeSettings(&server.ChangeSettingsParams{
						Body: server.ChangeSettingsBody{
							EnableStt: true,
						},
						Context: pmmapitests.Context,
					})
					require.NoError(t, err)
					assert.True(t, res.Payload.Settings.SttEnabled)
					assert.False(t, res.Payload.Settings.TelemetryEnabled)

					resg, err := serverClient.Default.Server.GetSettings(nil)
					require.NoError(t, err)
					assert.True(t, resg.Payload.Settings.SttEnabled)
					assert.False(t, resg.Payload.Settings.TelemetryEnabled)
				})

				t.Run("EnableTelemetryWhileItIsDisabled", func(t *testing.T) {
					defer restoreSettingsDefaults(t)

					res, err := serverClient.Default.Server.ChangeSettings(&server.ChangeSettingsParams{
						Body: server.ChangeSettingsBody{
							EnableTelemetry: true,
						},
						Context: pmmapitests.Context,
					})
					require.NoError(t, err)
					assert.True(t, res.Payload.Settings.TelemetryEnabled)

					resg, err := serverClient.Default.Server.GetSettings(nil)
					require.NoError(t, err)
					assert.True(t, resg.Payload.Settings.TelemetryEnabled)
					assert.True(t, resg.Payload.Settings.SttEnabled)
				})
			})

			t.Run("InvalidBothEnableAndDisableTelemetry", func(t *testing.T) {
				defer restoreSettingsDefaults(t)

				res, err := serverClient.Default.Server.ChangeSettings(&server.ChangeSettingsParams{
					Body: server.ChangeSettingsBody{
						EnableTelemetry:  true,
						DisableTelemetry: true,
					},
					Context: pmmapitests.Context,
				})
				pmmapitests.AssertAPIErrorf(t, err, 400, codes.InvalidArgument,
					`Invalid argument: both enable_telemetry and disable_telemetry are present.`)
				assert.Empty(t, res)
			})

			t.Run("InvalidPartition", func(t *testing.T) {
				defer restoreSettingsDefaults(t)

				res, err := serverClient.Default.Server.ChangeSettings(&server.ChangeSettingsParams{
					Body: server.ChangeSettingsBody{
						AWSPartitions: []string{"aws-123"},
					},
					Context: pmmapitests.Context,
				})
				pmmapitests.AssertAPIErrorf(t, err, 400, codes.InvalidArgument,
					`Invalid argument: aws_partitions: partition "aws-123" is invalid.`)
				assert.Empty(t, res)
			})

			t.Run("TooManyPartitions", func(t *testing.T) {
				defer restoreSettingsDefaults(t)

				res, err := serverClient.Default.Server.ChangeSettings(&server.ChangeSettingsParams{
					Body: server.ChangeSettingsBody{
						// We're expecting that 10 elements will be more than number of default partitions, which currently equals 6.
						AWSPartitions: []string{"aws", "aws", "aws", "aws", "aws", "aws", "aws", "aws", "aws", "aws"},
					},
					Context: pmmapitests.Context,
				})
				pmmapitests.AssertAPIErrorf(t, err, 400, codes.InvalidArgument,
					`Invalid argument: aws_partitions: list is too long.`)
				assert.Empty(t, res)
			})

			t.Run("HRInvalid", func(t *testing.T) {
				defer restoreSettingsDefaults(t)

				res, err := serverClient.Default.Server.ChangeSettings(&server.ChangeSettingsParams{
					Body: server.ChangeSettingsBody{
						MetricsResolutions: &server.ChangeSettingsParamsBodyMetricsResolutions{
							Hr: "1",
						},
					},
					Context: pmmapitests.Context,
				})
				pmmapitests.AssertAPIErrorf(t, err, 400, codes.InvalidArgument,
					`invalid google.protobuf.Duration value "1"`)
				assert.Empty(t, res)
			})

			t.Run("HRTooSmall", func(t *testing.T) {
				defer restoreSettingsDefaults(t)

				res, err := serverClient.Default.Server.ChangeSettings(&server.ChangeSettingsParams{
					Body: server.ChangeSettingsBody{
						MetricsResolutions: &server.ChangeSettingsParamsBodyMetricsResolutions{
							Hr: "0.5s",
						},
					},
					Context: pmmapitests.Context,
				})
				pmmapitests.AssertAPIErrorf(t, err, 400, codes.InvalidArgument,
					`Invalid argument: hr: minimal resolution is 1s.`)
				assert.Empty(t, res)
			})

			t.Run("HRFractional", func(t *testing.T) {
				defer restoreSettingsDefaults(t)

				res, err := serverClient.Default.Server.ChangeSettings(&server.ChangeSettingsParams{
					Body: server.ChangeSettingsBody{
						MetricsResolutions: &server.ChangeSettingsParamsBodyMetricsResolutions{
							Hr: "1.5s",
						},
					},
					Context: pmmapitests.Context,
				})
				pmmapitests.AssertAPIErrorf(t, err, 400, codes.InvalidArgument,
					`Invalid argument: hr: should be a natural number of seconds.`)
				assert.Empty(t, res)
			})

			t.Run("STTCheckIntervalInvalid", func(t *testing.T) {
				defer restoreSettingsDefaults(t)

				res, err := serverClient.Default.Server.ChangeSettings(&server.ChangeSettingsParams{
					Body: server.ChangeSettingsBody{
						SttCheckIntervals: &server.ChangeSettingsParamsBodySttCheckIntervals{
							FrequentInterval: "1",
						},
					},
					Context: pmmapitests.Context,
				})
				pmmapitests.AssertAPIErrorf(t, err, 400, codes.InvalidArgument,
					`invalid google.protobuf.Duration value "1"`)
				assert.Empty(t, res)
			})

			t.Run("SetPMMPublicAddressWithoutScheme", func(t *testing.T) {
				defer restoreSettingsDefaults(t)
				publicAddress := "192.168.0.42:8443"
				res, err := serverClient.Default.Server.ChangeSettings(&server.ChangeSettingsParams{
					Body: server.ChangeSettingsBody{
						PMMPublicAddress: publicAddress,
					},
					Context: pmmapitests.Context,
				})
				require.NoError(t, err)
				assert.Equal(t, publicAddress, res.Payload.Settings.PMMPublicAddress)
			})

			t.Run("STTCheckIntervalTooSmall", func(t *testing.T) {
				defer restoreSettingsDefaults(t)

				res, err := serverClient.Default.Server.ChangeSettings(&server.ChangeSettingsParams{
					Body: server.ChangeSettingsBody{
						SttCheckIntervals: &server.ChangeSettingsParamsBodySttCheckIntervals{
							StandardInterval: "0.9s",
						},
					},
					Context: pmmapitests.Context,
				})
				pmmapitests.AssertAPIErrorf(t, err, 400, codes.InvalidArgument,
					`Invalid argument: standard_interval: minimal resolution is 1s.`)
				assert.Empty(t, res)
			})

			t.Run("STTCheckIntervalFractional", func(t *testing.T) {
				defer restoreSettingsDefaults(t)

				res, err := serverClient.Default.Server.ChangeSettings(&server.ChangeSettingsParams{
					Body: server.ChangeSettingsBody{
						SttCheckIntervals: &server.ChangeSettingsParamsBodySttCheckIntervals{
							RareInterval: "1.5s",
						},
					},
					Context: pmmapitests.Context,
				})
				pmmapitests.AssertAPIErrorf(t, err, 400, codes.InvalidArgument,
					`Invalid argument: rare_interval: should be a natural number of seconds.`)
				assert.Empty(t, res)
			})

			t.Run("DataRetentionInvalid", func(t *testing.T) {
				defer restoreSettingsDefaults(t)

				res, err := serverClient.Default.Server.ChangeSettings(&server.ChangeSettingsParams{
					Body: server.ChangeSettingsBody{
						DataRetention: "1",
					},
					Context: pmmapitests.Context,
				})
				pmmapitests.AssertAPIErrorf(t, err, 400, codes.InvalidArgument,
					`invalid google.protobuf.Duration value "1"`)
				assert.Empty(t, res)
			})

			t.Run("DataRetentionInvalidToSmall", func(t *testing.T) {
				defer restoreSettingsDefaults(t)

				res, err := serverClient.Default.Server.ChangeSettings(&server.ChangeSettingsParams{
					Body: server.ChangeSettingsBody{
						DataRetention: "10s",
					},
					Context: pmmapitests.Context,
				})
				pmmapitests.AssertAPIErrorf(t, err, 400, codes.InvalidArgument,
					`Invalid argument: data_retention: minimal resolution is 24h.`)
				assert.Empty(t, res)
			})

			t.Run("DataRetentionFractional", func(t *testing.T) {
				defer restoreSettingsDefaults(t)

				res, err := serverClient.Default.Server.ChangeSettings(&server.ChangeSettingsParams{
					Body: server.ChangeSettingsBody{
						DataRetention: "36h",
					},
					Context: pmmapitests.Context,
				})
				pmmapitests.AssertAPIErrorf(t, err, 400, codes.InvalidArgument,
					`invalid google.protobuf.Duration value "36h"`)
				assert.Empty(t, res)
			})

			t.Run("InvalidSSHKey", func(t *testing.T) {
				defer restoreSettingsDefaults(t)

				res, err := serverClient.Default.Server.ChangeSettings(&server.ChangeSettingsParams{
					Body: server.ChangeSettingsBody{
						SSHKey: "some-invalid-ssh-key",
					},
					Context: pmmapitests.Context,
				})
				pmmapitests.AssertAPIErrorf(t, err, 400, codes.InvalidArgument, `Invalid SSH key.`)
				assert.Empty(t, res)
			})

			t.Run("NoAdminUserForSSH", func(t *testing.T) {
				defer restoreSettingsDefaults(t)

				sshKey := "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQClY/8sz3w03vA2bY6mBFgUzrvb2FIoHw8ZjUXGGClJzJg5HC" +
					"3jW1m5df7TOIkx0bt6Da2UOhuCvS4o27IT1aiHXVFydppp6ghQRB6saiiW2TKlQ7B+mXatwVaOIkO381kEjgijAs0LJn" +
					"NRGpqQW0ZEAxVMz4a8puaZmVNicYSVYs4kV3QZsHuqn7jHbxs5NGAO+uRRSjcuPXregsyd87RAUHkGmNrwNFln/XddMz" +
					"dGMwqZOuZWuxIXBqSrSX927XGHAJlUaOmLz5etZXHzfAY1Zxfu39r66Sx95bpm3JBmc/Ewfr8T2WL0cqynkpH+3QQBCj" +
					"weTHzBE+lpXHdR2se1 qsandbox"
				res, err := serverClient.Default.Server.ChangeSettings(&server.ChangeSettingsParams{
					Body: server.ChangeSettingsBody{
						SSHKey: sshKey,
					},
					Context: pmmapitests.Context,
				})
				pmmapitests.AssertAPIErrorf(t, err, 500, codes.Internal, `Internal server error.`)
				assert.Empty(t, res)
			})

			t.Run("OK", func(t *testing.T) {
				defer restoreSettingsDefaults(t)

				res, err := serverClient.Default.Server.ChangeSettings(&server.ChangeSettingsParams{
					Body: server.ChangeSettingsBody{
						DisableTelemetry: true,
						MetricsResolutions: &server.ChangeSettingsParamsBodyMetricsResolutions{
							Hr: "2s",
							Mr: "15s",
							Lr: "120s", // 2 minutes
						},
						DataRetention: "864000s",                           // 240 hours
						AWSPartitions: []string{"aws-cn", "aws", "aws-cn"}, // duplicates are ok
					},
					Context: pmmapitests.Context,
				})
				require.NoError(t, err)
				assert.False(t, res.Payload.Settings.TelemetryEnabled)
				expected := &server.ChangeSettingsOKBodySettingsMetricsResolutions{
					Hr: "2s",
					Mr: "15s",
					Lr: "120s",
				}
				assert.Equal(t, expected, res.Payload.Settings.MetricsResolutions)
				assert.Equal(t, []string{"aws", "aws-cn"}, res.Payload.Settings.AWSPartitions)

				getRes, err := serverClient.Default.Server.GetSettings(nil)
				require.NoError(t, err)
				assert.False(t, getRes.Payload.Settings.TelemetryEnabled)
				getExpected := &server.GetSettingsOKBodySettingsMetricsResolutions{
					Hr: "2s",
					Mr: "15s",
					Lr: "120s",
				}
				assert.Equal(t, getExpected, getRes.Payload.Settings.MetricsResolutions)
				assert.Equal(t, "864000s", res.Payload.Settings.DataRetention)
				assert.Equal(t, []string{"aws", "aws-cn"}, res.Payload.Settings.AWSPartitions)

				t.Run("DefaultsAreNotRestored", func(t *testing.T) {
					defer restoreSettingsDefaults(t)

					res, err := serverClient.Default.Server.ChangeSettings(&server.ChangeSettingsParams{
						Body:    server.ChangeSettingsBody{},
						Context: pmmapitests.Context,
					})
					require.NoError(t, err)
					assert.False(t, res.Payload.Settings.TelemetryEnabled)
					expected := &server.ChangeSettingsOKBodySettingsMetricsResolutions{
						Hr: "2s",
						Mr: "15s",
						Lr: "120s",
					}
					assert.Equal(t, expected, res.Payload.Settings.MetricsResolutions)
					assert.Equal(t, []string{"aws", "aws-cn"}, res.Payload.Settings.AWSPartitions)

					// Check if the values were persisted
					getRes, err := serverClient.Default.Server.GetSettings(nil)
					require.NoError(t, err)
					assert.False(t, getRes.Payload.Settings.TelemetryEnabled)
					getExpected := &server.GetSettingsOKBodySettingsMetricsResolutions{
						Hr: "2s",
						Mr: "15s",
						Lr: "120s",
					}
					assert.Equal(t, getExpected, getRes.Payload.Settings.MetricsResolutions)
					assert.Equal(t, "864000s", res.Payload.Settings.DataRetention)
					assert.Equal(t, []string{"aws", "aws-cn"}, res.Payload.Settings.AWSPartitions)
				})
			})

			t.Run("STTCheckIntervalsValid", func(t *testing.T) {
				defer restoreSettingsDefaults(t)

				res, err := serverClient.Default.Server.ChangeSettings(&server.ChangeSettingsParams{
					Body: server.ChangeSettingsBody{
						SttCheckIntervals: &server.ChangeSettingsParamsBodySttCheckIntervals{
							RareInterval:     "28800s", // 8 hours
							StandardInterval: "1800s",  // 30 minutes
							FrequentInterval: "20s",
						},
					},
					Context: pmmapitests.Context,
				})
				require.NoError(t, err)
				expected := &server.ChangeSettingsOKBodySettingsSttCheckIntervals{
					RareInterval:     "28800s",
					StandardInterval: "1800s",
					FrequentInterval: "20s",
				}
				assert.Equal(t, expected, res.Payload.Settings.SttCheckIntervals)

				getRes, err := serverClient.Default.Server.GetSettings(nil)
				require.NoError(t, err)
				getExpected := &server.GetSettingsOKBodySettingsSttCheckIntervals{
					RareInterval:     "28800s",
					StandardInterval: "1800s",
					FrequentInterval: "20s",
				}
				assert.Equal(t, getExpected, getRes.Payload.Settings.SttCheckIntervals)

				t.Run("DefaultsAreNotRestored", func(t *testing.T) {
					defer restoreSettingsDefaults(t)

					res, err := serverClient.Default.Server.ChangeSettings(&server.ChangeSettingsParams{
						Body:    server.ChangeSettingsBody{},
						Context: pmmapitests.Context,
					})
					require.NoError(t, err)
					expected := &server.ChangeSettingsOKBodySettingsSttCheckIntervals{
						RareInterval:     "28800s",
						StandardInterval: "1800s",
						FrequentInterval: "20s",
					}
					assert.Equal(t, expected, res.Payload.Settings.SttCheckIntervals)

					// Check if the values were persisted
					getRes, err := serverClient.Default.Server.GetSettings(nil)
					require.NoError(t, err)
					getExpected := &server.GetSettingsOKBodySettingsSttCheckIntervals{
						RareInterval:     "28800s",
						StandardInterval: "1800s",
						FrequentInterval: "20s",
					}
					assert.Equal(t, getExpected, getRes.Payload.Settings.SttCheckIntervals)
				})
			})

			t.Run("AlertManager", func(t *testing.T) {
				t.Run("SetInvalid", func(t *testing.T) {
					defer restoreSettingsDefaults(t)

					url := "http://localhost:1234/"
					rules := `invalid rules`

					_, err := serverClient.Default.Server.ChangeSettings(&server.ChangeSettingsParams{
						Body: server.ChangeSettingsBody{
							AlertManagerURL:   url,
							AlertManagerRules: rules,
						},
						Context: pmmapitests.Context,
					})
					pmmapitests.AssertAPIErrorf(t, err, 400, codes.InvalidArgument, `Invalid alerting rules.`)

					gets, err := serverClient.Default.Server.GetSettings(nil)
					require.NoError(t, err)
					assert.Empty(t, gets.Payload.Settings.AlertManagerURL)
					assert.Empty(t, gets.Payload.Settings.AlertManagerRules)
				})

				t.Run("SetAndRemoveInvalid", func(t *testing.T) {
					defer restoreSettingsDefaults(t)

					_, err := serverClient.Default.Server.ChangeSettings(&server.ChangeSettingsParams{
						Body: server.ChangeSettingsBody{
							AlertManagerURL:       "invalid url",
							RemoveAlertManagerURL: true,
						},
						Context: pmmapitests.Context,
					})
					pmmapitests.AssertAPIErrorf(t, err, 400, codes.InvalidArgument,
						`Invalid argument: both alert_manager_url and remove_alert_manager_url are present.`)

					gets, err := serverClient.Default.Server.GetSettings(nil)
					require.NoError(t, err)
					assert.Empty(t, gets.Payload.Settings.AlertManagerURL)
					assert.Empty(t, gets.Payload.Settings.AlertManagerRules)
				})

				t.Run("SetValid", func(t *testing.T) {
					defer restoreSettingsDefaults(t)

					url := "http://localhost:1234/"
					rules := strings.TrimSpace(`
groups:
- name: example
  rules:
  - alert: HighRequestLatency
    expr: job:request_latency_seconds:mean5m{job="myjob"} > 0.5
    for: 10m
    labels:
      severity: page
    annotations:
      summary: High request latency
					`) + "\n"

					res, err := serverClient.Default.Server.ChangeSettings(&server.ChangeSettingsParams{
						Body: server.ChangeSettingsBody{
							AlertManagerURL:   url,
							AlertManagerRules: rules,
						},
						Context: pmmapitests.Context,
					})
					require.NoError(t, err)
					assert.Equal(t, url, res.Payload.Settings.AlertManagerURL)
					assert.Equal(t, rules, res.Payload.Settings.AlertManagerRules)

					gets, err := serverClient.Default.Server.GetSettings(nil)
					require.NoError(t, err)
					assert.Equal(t, url, gets.Payload.Settings.AlertManagerURL)
					assert.Equal(t, rules, gets.Payload.Settings.AlertManagerRules)

					t.Run("EmptyShouldNotRemove", func(t *testing.T) {
						defer restoreSettingsDefaults(t)

						_, err := serverClient.Default.Server.ChangeSettings(&server.ChangeSettingsParams{
							Body:    server.ChangeSettingsBody{},
							Context: pmmapitests.Context,
						})
						require.NoError(t, err)

						gets, err = serverClient.Default.Server.GetSettings(nil)
						require.NoError(t, err)
						assert.Equal(t, url, gets.Payload.Settings.AlertManagerURL)
						assert.Equal(t, rules, gets.Payload.Settings.AlertManagerRules)
					})
				})
			})

			t.Run("grpc-gateway", func(t *testing.T) {
				// Test with pure JSON without swagger for tracking grpc-gateway behavior:
				// https://github.com/grpc-ecosystem/grpc-gateway/issues/400

				// do not use generated types as they can do extra work in generated methods
				type params struct {
					Settings struct {
						MetricsResolutions struct {
							LR string `json:"lr"`
						} `json:"metrics_resolutions"`
					} `json:"settings"`
				}
				changeURI := pmmapitests.BaseURL.ResolveReference(&url.URL{
					Path: "v1/Settings/Change",
				})
				getURI := pmmapitests.BaseURL.ResolveReference(&url.URL{
					Path: "v1/Settings/Get",
				})

				for change, get := range map[string]string{
					"59s": "59s",
					"60s": "60s",
					"61s": "61s",
					"61":  "", // no suffix => error
					"2m":  "", // m suffix => error
					"1h":  "", // h suffix => error
					"1d":  "", // d suffix => error
					"1w":  "", // w suffix => error
				} {
					change, get := change, get
					t.Run(change, func(t *testing.T) {
						defer restoreSettingsDefaults(t)

						var p params
						p.Settings.MetricsResolutions.LR = change
						b, err := json.Marshal(p.Settings)
						require.NoError(t, err)
						req, err := http.NewRequestWithContext(pmmapitests.Context, http.MethodPost, changeURI.String(), bytes.NewReader(b))
						require.NoError(t, err)
						if pmmapitests.Debug {
							b, err = httputil.DumpRequestOut(req, true)
							require.NoError(t, err)
							t.Logf("Request:\n%s", b)
						}

						resp, err := http.DefaultClient.Do(req)
						require.NoError(t, err)
						if pmmapitests.Debug {
							b, err = httputil.DumpResponse(resp, true)
							require.NoError(t, err)
							t.Logf("Response:\n%s", b)
						}
						b, err = io.ReadAll(resp.Body)
						assert.NoError(t, err)
						resp.Body.Close() //nolint:errcheck

						if get == "" {
							assert.Equal(t, 400, resp.StatusCode, "response:\n%s", b)
							return
						}
						assert.Equal(t, 200, resp.StatusCode, "response:\n%s", b)

						p.Settings.MetricsResolutions.LR = ""
						err = json.Unmarshal(b, &p)
						require.NoError(t, err)
						assert.Equal(t, get, p.Settings.MetricsResolutions.LR, "Change")

						req, err = http.NewRequestWithContext(pmmapitests.Context, http.MethodPost, getURI.String(), nil)
						require.NoError(t, err)
						if pmmapitests.Debug {
							b, err = httputil.DumpRequestOut(req, true)
							require.NoError(t, err)
							t.Logf("Request:\n%s", b)
						}

						resp, err = http.DefaultClient.Do(req)
						require.NoError(t, err)
						if pmmapitests.Debug {
							b, err = httputil.DumpResponse(resp, true)
							require.NoError(t, err)
							t.Logf("Response:\n%s", b)
						}
						b, err = io.ReadAll(resp.Body)
						assert.NoError(t, err)
						resp.Body.Close() //nolint:errcheck
						assert.Equal(t, 200, resp.StatusCode, "response:\n%s", b)

						p.Settings.MetricsResolutions.LR = ""
						err = json.Unmarshal(b, &p)
						require.NoError(t, err)
						assert.Equal(t, get, p.Settings.MetricsResolutions.LR, "Get")
					})
				}
			})
		})
	})
}
