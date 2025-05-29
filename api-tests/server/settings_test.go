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
	"testing"

	"github.com/AlekSi/pointer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"

	pmmapitests "github.com/percona/pmm/api-tests"
	serverClient "github.com/percona/pmm/api/server/v1/json/client"
	server "github.com/percona/pmm/api/server/v1/json/client/server_service"
)

func TestSettings(t *testing.T) {
	t.Run("GetSettings", func(t *testing.T) {
		res, err := serverClient.Default.ServerService.GetSettings(nil)
		require.NoError(t, err)
		assert.True(t, res.Payload.Settings.TelemetryEnabled)
		assert.True(t, res.Payload.Settings.AdvisorEnabled)
		expected := &server.GetSettingsOKBodySettingsMetricsResolutions{
			Hr: "5s",
			Mr: "10s",
			Lr: "60s",
		}
		assert.Equal(t, expected, res.Payload.Settings.MetricsResolutions)
		expectedAdvisorRunIntervals := &server.GetSettingsOKBodySettingsAdvisorRunIntervals{
			FrequentInterval: "14400s",
			StandardInterval: "86400s",
			RareInterval:     "280800s",
		}
		assert.Equal(t, expectedAdvisorRunIntervals, res.Payload.Settings.AdvisorRunIntervals)
		assert.Equal(t, "2592000s", res.Payload.Settings.DataRetention)
		assert.Equal(t, []string{"aws"}, res.Payload.Settings.AWSPartitions)
		assert.True(t, res.Payload.Settings.UpdatesEnabled)
		assert.True(t, res.Payload.Settings.AlertingEnabled)

		t.Run("ChangeSettings", func(t *testing.T) {
			defer restoreSettingsDefaults(t)

			t.Run("Updates", func(t *testing.T) {
				t.Run("DisableAndEnableUpdatesSettingsUpdate", func(t *testing.T) {
					defer restoreSettingsDefaults(t)
					res, err := serverClient.Default.ServerService.ChangeSettings(&server.ChangeSettingsParams{
						Body: server.ChangeSettingsBody{
							EnableUpdates: pointer.ToBool(false),
						},
						Context: pmmapitests.Context,
					})
					require.NoError(t, err)
					assert.False(t, res.Payload.Settings.UpdatesEnabled)

					resg, err := serverClient.Default.ServerService.GetSettings(nil)
					require.NoError(t, err)
					assert.False(t, resg.Payload.Settings.UpdatesEnabled)

					res, err = serverClient.Default.ServerService.ChangeSettings(&server.ChangeSettingsParams{
						Body: server.ChangeSettingsBody{
							EnableUpdates: pointer.ToBool(true),
						},
						Context: pmmapitests.Context,
					})
					require.NoError(t, err)
					assert.True(t, res.Payload.Settings.UpdatesEnabled)

					resg, err = serverClient.Default.ServerService.GetSettings(nil)
					require.NoError(t, err)
					assert.True(t, resg.Payload.Settings.UpdatesEnabled)
				})
			})

			t.Run("ValidAlertingSettingsUpdate", func(t *testing.T) {
				defer restoreSettingsDefaults(t)

				res, err := serverClient.Default.ServerService.ChangeSettings(&server.ChangeSettingsParams{
					Body: server.ChangeSettingsBody{
						EnableAlerting: pointer.ToBool(false),
					},
					Context: pmmapitests.Context,
				})
				require.NoError(t, err)
				assert.False(t, res.Payload.Settings.AlertingEnabled)
			})

			t.Run("EnableAdviorsAndEnableTelemetry", func(t *testing.T) {
				defer restoreSettingsDefaults(t)

				res, err := serverClient.Default.ServerService.ChangeSettings(&server.ChangeSettingsParams{
					Body: server.ChangeSettingsBody{
						EnableAdvisor:   pointer.ToBool(true),
						EnableTelemetry: pointer.ToBool(true),
					},
					Context: pmmapitests.Context,
				})
				require.NoError(t, err)
				assert.True(t, res.Payload.Settings.AdvisorEnabled)
				assert.True(t, res.Payload.Settings.TelemetryEnabled)

				resg, err := serverClient.Default.ServerService.GetSettings(nil)
				require.NoError(t, err)
				assert.True(t, resg.Payload.Settings.TelemetryEnabled)
				assert.True(t, resg.Payload.Settings.AdvisorEnabled)
			})

			t.Run("EnableAdvisorsAndDisableTelemetry", func(t *testing.T) {
				defer restoreSettingsDefaults(t)

				res, err := serverClient.Default.ServerService.ChangeSettings(&server.ChangeSettingsParams{
					Body: server.ChangeSettingsBody{
						EnableAdvisor:   pointer.ToBool(true),
						EnableTelemetry: pointer.ToBool(false),
					},
					Context: pmmapitests.Context,
				})
				require.NoError(t, err)
				assert.True(t, res.Payload.Settings.AdvisorEnabled)
				assert.False(t, res.Payload.Settings.TelemetryEnabled)
			})

			t.Run("DisableAdvisorsAndEnableTelemetry", func(t *testing.T) {
				defer restoreSettingsDefaults(t)

				res, err := serverClient.Default.ServerService.ChangeSettings(&server.ChangeSettingsParams{
					Body: server.ChangeSettingsBody{
						EnableAdvisor:   pointer.ToBool(false),
						EnableTelemetry: pointer.ToBool(true),
					},
					Context: pmmapitests.Context,
				})
				require.NoError(t, err)
				assert.False(t, res.Payload.Settings.AdvisorEnabled)
				assert.True(t, res.Payload.Settings.TelemetryEnabled)

				resg, err := serverClient.Default.ServerService.GetSettings(nil)
				require.NoError(t, err)
				assert.True(t, resg.Payload.Settings.TelemetryEnabled)
				assert.False(t, resg.Payload.Settings.AdvisorEnabled)
			})

			t.Run("DisableAdvisorsAndDisableTelemetry", func(t *testing.T) {
				defer restoreSettingsDefaults(t)

				res, err := serverClient.Default.ServerService.ChangeSettings(&server.ChangeSettingsParams{
					Body: server.ChangeSettingsBody{
						EnableAdvisor:   pointer.ToBool(false),
						EnableTelemetry: pointer.ToBool(false),
					},
					Context: pmmapitests.Context,
				})
				require.NoError(t, err)
				assert.False(t, res.Payload.Settings.AdvisorEnabled)
				assert.False(t, res.Payload.Settings.TelemetryEnabled)

				resg, err := serverClient.Default.ServerService.GetSettings(nil)
				require.NoError(t, err)
				assert.False(t, resg.Payload.Settings.TelemetryEnabled)
				assert.False(t, resg.Payload.Settings.AdvisorEnabled)
			})

			t.Run("EnableAdvisorsWhileTelemetryEnabled", func(t *testing.T) {
				defer restoreSettingsDefaults(t)

				// Ensure Telemetry is enabled
				res, err := serverClient.Default.ServerService.ChangeSettings(&server.ChangeSettingsParams{
					Body: server.ChangeSettingsBody{
						EnableTelemetry: pointer.ToBool(true),
					},
					Context: pmmapitests.Context,
				})
				require.NoError(t, err)
				assert.True(t, res.Payload.Settings.TelemetryEnabled)

				res, err = serverClient.Default.ServerService.ChangeSettings(&server.ChangeSettingsParams{
					Body: server.ChangeSettingsBody{
						EnableAdvisor: pointer.ToBool(true),
					},
					Context: pmmapitests.Context,
				})
				require.NoError(t, err)
				assert.True(t, res.Payload.Settings.AdvisorEnabled)

				resg, err := serverClient.Default.ServerService.GetSettings(nil)
				require.NoError(t, err)
				assert.True(t, resg.Payload.Settings.TelemetryEnabled)
				assert.True(t, resg.Payload.Settings.AdvisorEnabled)
			})

			t.Run("DisableAdvisorsWhileItIsDisabled", func(t *testing.T) {
				defer restoreSettingsDefaults(t)

				res, err := serverClient.Default.ServerService.ChangeSettings(&server.ChangeSettingsParams{
					Body: server.ChangeSettingsBody{
						EnableAdvisor: pointer.ToBool(false),
					},
					Context: pmmapitests.Context,
				})
				require.NoError(t, err)
				assert.False(t, res.Payload.Settings.AdvisorEnabled)

				resg, err := serverClient.Default.ServerService.GetSettings(nil)
				require.NoError(t, err)
				assert.True(t, resg.Payload.Settings.TelemetryEnabled)
				assert.False(t, resg.Payload.Settings.AdvisorEnabled)
			})

			t.Run("AdvisorsEnabledState", func(t *testing.T) {
				defer restoreSettingsDefaults(t)

				res, err := serverClient.Default.ServerService.ChangeSettings(&server.ChangeSettingsParams{
					Body: server.ChangeSettingsBody{
						EnableAdvisor: pointer.ToBool(true),
					},
					Context: pmmapitests.Context,
				})

				require.NoError(t, err)
				assert.True(t, res.Payload.Settings.AdvisorEnabled)

				resg, err := serverClient.Default.ServerService.GetSettings(nil)
				require.NoError(t, err)
				assert.True(t, resg.Payload.Settings.TelemetryEnabled)
				assert.True(t, resg.Payload.Settings.AdvisorEnabled)

				t.Run("EnableAdvisorsWhileItIsEnabled", func(t *testing.T) {
					res, err := serverClient.Default.ServerService.ChangeSettings(&server.ChangeSettingsParams{
						Body: server.ChangeSettingsBody{
							EnableAdvisor: pointer.ToBool(true),
						},
						Context: pmmapitests.Context,
					})
					require.NoError(t, err)
					assert.True(t, res.Payload.Settings.AdvisorEnabled)

					resg, err := serverClient.Default.ServerService.GetSettings(nil)
					require.NoError(t, err)
					assert.True(t, resg.Payload.Settings.TelemetryEnabled)
					assert.True(t, resg.Payload.Settings.AdvisorEnabled)
				})

				t.Run("DisableTelemetryWhileAdvisorsEnabled", func(t *testing.T) {
					res, err := serverClient.Default.ServerService.ChangeSettings(&server.ChangeSettingsParams{
						Body: server.ChangeSettingsBody{
							EnableTelemetry: pointer.ToBool(false),
						},
						Context: pmmapitests.Context,
					})
					require.NoError(t, err)
					assert.True(t, res.Payload.Settings.AdvisorEnabled)
					assert.False(t, res.Payload.Settings.TelemetryEnabled)
				})
			})

			t.Run("TelemetryDisabledState", func(t *testing.T) {
				defer restoreSettingsDefaults(t)

				res, err := serverClient.Default.ServerService.ChangeSettings(&server.ChangeSettingsParams{
					Body: server.ChangeSettingsBody{
						EnableTelemetry: pointer.ToBool(false),
					},
					Context: pmmapitests.Context,
				})

				require.NoError(t, err)
				assert.False(t, res.Payload.Settings.TelemetryEnabled)

				resg, err := serverClient.Default.ServerService.GetSettings(nil)
				require.NoError(t, err)
				assert.False(t, resg.Payload.Settings.TelemetryEnabled)
				assert.True(t, resg.Payload.Settings.AdvisorEnabled)

				t.Run("EnableAdvisorsWhileTelemetryDisabled", func(t *testing.T) {
					defer restoreSettingsDefaults(t)

					res, err := serverClient.Default.ServerService.ChangeSettings(&server.ChangeSettingsParams{
						Body: server.ChangeSettingsBody{
							EnableAdvisor: pointer.ToBool(true),
						},
						Context: pmmapitests.Context,
					})
					require.NoError(t, err)
					assert.True(t, res.Payload.Settings.AdvisorEnabled)
					assert.False(t, res.Payload.Settings.TelemetryEnabled)

					resg, err := serverClient.Default.ServerService.GetSettings(nil)
					require.NoError(t, err)
					assert.True(t, resg.Payload.Settings.AdvisorEnabled)
					assert.False(t, resg.Payload.Settings.TelemetryEnabled)
				})

				t.Run("EnableTelemetryWhileItIsDisabled", func(t *testing.T) {
					defer restoreSettingsDefaults(t)

					res, err := serverClient.Default.ServerService.ChangeSettings(&server.ChangeSettingsParams{
						Body: server.ChangeSettingsBody{
							EnableTelemetry: pointer.ToBool(true),
						},
						Context: pmmapitests.Context,
					})
					require.NoError(t, err)
					assert.True(t, res.Payload.Settings.TelemetryEnabled)

					resg, err := serverClient.Default.ServerService.GetSettings(nil)
					require.NoError(t, err)
					assert.True(t, resg.Payload.Settings.TelemetryEnabled)
					assert.True(t, resg.Payload.Settings.AdvisorEnabled)
				})
			})

			t.Run("InvalidPartition", func(t *testing.T) {
				defer restoreSettingsDefaults(t)

				res, err := serverClient.Default.ServerService.ChangeSettings(&server.ChangeSettingsParams{
					Body: server.ChangeSettingsBody{
						AWSPartitions: &server.ChangeSettingsParamsBodyAWSPartitions{Values: []string{"aws-123"}},
					},
					Context: pmmapitests.Context,
				})
				pmmapitests.AssertAPIErrorf(t, err, 400, codes.InvalidArgument,
					`Invalid argument: aws_partitions: partition "aws-123" is invalid.`)
				assert.Empty(t, res)
			})

			t.Run("TooManyPartitions", func(t *testing.T) {
				defer restoreSettingsDefaults(t)

				res, err := serverClient.Default.ServerService.ChangeSettings(&server.ChangeSettingsParams{
					Body: server.ChangeSettingsBody{
						// We're expecting that 10 elements will be more than number of default partitions, which currently equals 6.
						AWSPartitions: &server.ChangeSettingsParamsBodyAWSPartitions{Values: []string{"aws", "aws", "aws", "aws", "aws", "aws", "aws", "aws", "aws", "aws"}},
					},
					Context: pmmapitests.Context,
				})
				pmmapitests.AssertAPIErrorf(t, err, 400, codes.InvalidArgument,
					`Invalid argument: aws_partitions: list is too long.`)
				assert.Empty(t, res)
			})

			t.Run("HRInvalid", func(t *testing.T) {
				defer restoreSettingsDefaults(t)

				res, err := serverClient.Default.ServerService.ChangeSettings(&server.ChangeSettingsParams{
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

				res, err := serverClient.Default.ServerService.ChangeSettings(&server.ChangeSettingsParams{
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

				res, err := serverClient.Default.ServerService.ChangeSettings(&server.ChangeSettingsParams{
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

			t.Run("AdvisorsCheckIntervalInvalid", func(t *testing.T) {
				defer restoreSettingsDefaults(t)

				res, err := serverClient.Default.ServerService.ChangeSettings(&server.ChangeSettingsParams{
					Body: server.ChangeSettingsBody{
						AdvisorRunIntervals: &server.ChangeSettingsParamsBodyAdvisorRunIntervals{
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
				res, err := serverClient.Default.ServerService.ChangeSettings(&server.ChangeSettingsParams{
					Body: server.ChangeSettingsBody{
						PMMPublicAddress: pointer.ToString(publicAddress),
					},
					Context: pmmapitests.Context,
				})
				require.NoError(t, err)
				assert.Equal(t, publicAddress, res.Payload.Settings.PMMPublicAddress)
			})

			t.Run("AdvisorsCheckIntervalTooSmall", func(t *testing.T) {
				defer restoreSettingsDefaults(t)

				res, err := serverClient.Default.ServerService.ChangeSettings(&server.ChangeSettingsParams{
					Body: server.ChangeSettingsBody{
						AdvisorRunIntervals: &server.ChangeSettingsParamsBodyAdvisorRunIntervals{
							StandardInterval: "0.9s",
						},
					},
					Context: pmmapitests.Context,
				})
				pmmapitests.AssertAPIErrorf(t, err, 400, codes.InvalidArgument,
					`Invalid argument: standard_interval: minimal resolution is 1s.`)
				assert.Empty(t, res)
			})

			t.Run("AdviorsCheckIntervalFractional", func(t *testing.T) {
				defer restoreSettingsDefaults(t)

				res, err := serverClient.Default.ServerService.ChangeSettings(&server.ChangeSettingsParams{
					Body: server.ChangeSettingsBody{
						AdvisorRunIntervals: &server.ChangeSettingsParamsBodyAdvisorRunIntervals{
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

				res, err := serverClient.Default.ServerService.ChangeSettings(&server.ChangeSettingsParams{
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

				res, err := serverClient.Default.ServerService.ChangeSettings(&server.ChangeSettingsParams{
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

				res, err := serverClient.Default.ServerService.ChangeSettings(&server.ChangeSettingsParams{
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

				res, err := serverClient.Default.ServerService.ChangeSettings(&server.ChangeSettingsParams{
					Body: server.ChangeSettingsBody{
						SSHKey: pointer.ToString("some-invalid-ssh-key"),
					},
					Context: pmmapitests.Context,
				})
				pmmapitests.AssertAPIErrorf(t, err, 400, codes.InvalidArgument, `Invalid SSH key.`)
				assert.Empty(t, res)
			})

			t.Run("ChangeSSHKey only on AMI and OVF", func(t *testing.T) {
				defer restoreSettingsDefaults(t)

				sshKey := "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQClY/8sz3w03vA2bY6mBFgUzrvb2FIoHw8ZjUXGGClJzJg5HC" +
					"3jW1m5df7TOIkx0bt6Da2UOhuCvS4o27IT1aiHXVFydppp6ghQRB6saiiW2TKlQ7B+mXatwVaOIkO381kEjgijAs0LJn" +
					"NRGpqQW0ZEAxVMz4a8puaZmVNicYSVYs4kV3QZsHuqn7jHbxs5NGAO+uRRSjcuPXregsyd87RAUHkGmNrwNFln/XddMz" +
					"dGMwqZOuZWuxIXBqSrSX927XGHAJlUaOmLz5etZXHzfAY1Zxfu39r66Sx95bpm3JBmc/Ewfr8T2WL0cqynkpH+3QQBCj" +
					"weTHzBE+lpXHdR2se1 qsandbox"
				res, err := serverClient.Default.ServerService.ChangeSettings(&server.ChangeSettingsParams{
					Body: server.ChangeSettingsBody{
						SSHKey: pointer.ToString(sshKey),
					},
					Context: pmmapitests.Context,
				})
				pmmapitests.AssertAPIErrorf(t, err, 500, codes.Internal, `SSH key can be set only on AMI and OVF distributions`)
				assert.Empty(t, res)
			})

			t.Run("OK", func(t *testing.T) {
				defer restoreSettingsDefaults(t)

				res, err := serverClient.Default.ServerService.ChangeSettings(&server.ChangeSettingsParams{
					Body: server.ChangeSettingsBody{
						EnableTelemetry: pointer.ToBool(false),
						MetricsResolutions: &server.ChangeSettingsParamsBodyMetricsResolutions{
							Hr: "2s",
							Mr: "15s",
							Lr: "120s", // 2 minutes
						},
						DataRetention: "864000s",                                                                                  // 240 hours
						AWSPartitions: &server.ChangeSettingsParamsBodyAWSPartitions{Values: []string{"aws-cn", "aws", "aws-cn"}}, // duplicates are ok
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

				getRes, err := serverClient.Default.ServerService.GetSettings(nil)
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

					res, err := serverClient.Default.ServerService.ChangeSettings(&server.ChangeSettingsParams{
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
					getRes, err := serverClient.Default.ServerService.GetSettings(nil)
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

			t.Run("AdvisorCheckIntervalsValid", func(t *testing.T) {
				defer restoreSettingsDefaults(t)

				res, err := serverClient.Default.ServerService.ChangeSettings(&server.ChangeSettingsParams{
					Body: server.ChangeSettingsBody{
						AdvisorRunIntervals: &server.ChangeSettingsParamsBodyAdvisorRunIntervals{
							RareInterval:     "28800s", // 8 hours
							StandardInterval: "1800s",  // 30 minutes
							FrequentInterval: "20s",
						},
					},
					Context: pmmapitests.Context,
				})
				require.NoError(t, err)
				expected := &server.ChangeSettingsOKBodySettingsAdvisorRunIntervals{
					RareInterval:     "28800s",
					StandardInterval: "1800s",
					FrequentInterval: "20s",
				}
				assert.Equal(t, expected, res.Payload.Settings.AdvisorRunIntervals)

				getRes, err := serverClient.Default.ServerService.GetSettings(nil)
				require.NoError(t, err)
				getExpected := &server.GetSettingsOKBodySettingsAdvisorRunIntervals{
					RareInterval:     "28800s",
					StandardInterval: "1800s",
					FrequentInterval: "20s",
				}
				assert.Equal(t, getExpected, getRes.Payload.Settings.AdvisorRunIntervals)

				t.Run("DefaultsAreNotRestored", func(t *testing.T) {
					defer restoreSettingsDefaults(t)

					res, err := serverClient.Default.ServerService.ChangeSettings(&server.ChangeSettingsParams{
						Body:    server.ChangeSettingsBody{},
						Context: pmmapitests.Context,
					})
					require.NoError(t, err)
					expected := &server.ChangeSettingsOKBodySettingsAdvisorRunIntervals{
						RareInterval:     "28800s",
						StandardInterval: "1800s",
						FrequentInterval: "20s",
					}
					assert.Equal(t, expected, res.Payload.Settings.AdvisorRunIntervals)

					// Check if the values were persisted
					getRes, err := serverClient.Default.ServerService.GetSettings(nil)
					require.NoError(t, err)
					getExpected := &server.GetSettingsOKBodySettingsAdvisorRunIntervals{
						RareInterval:     "28800s",
						StandardInterval: "1800s",
						FrequentInterval: "20s",
					}
					assert.Equal(t, getExpected, getRes.Payload.Settings.AdvisorRunIntervals)
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
					Path: "v1/server/settings",
				})
				getURI := pmmapitests.BaseURL.ResolveReference(&url.URL{
					Path: "v1/server/settings",
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
					t.Run(change, func(t *testing.T) {
						defer restoreSettingsDefaults(t)

						var p params
						p.Settings.MetricsResolutions.LR = change
						b, err := json.Marshal(p.Settings)
						require.NoError(t, err)
						req, err := http.NewRequestWithContext(pmmapitests.Context, http.MethodPut, changeURI.String(), bytes.NewReader(b))
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

						req, err = http.NewRequestWithContext(pmmapitests.Context, http.MethodGet, getURI.String(), nil)
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
