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

package ia

import (
	"testing"

	"github.com/brianvoe/gofakeit/v6"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"

	pmmapitests "github.com/percona/pmm/api-tests"
	alertingClient "github.com/percona/pmm/api/managementpb/alerting/json/client"
	channelsClient "github.com/percona/pmm/api/managementpb/ia/json/client"
	"github.com/percona/pmm/api/managementpb/ia/json/client/channels"
)

// Note: Even though the IA services check for alerting enabled or disabled before returning results
// we don't enable or disable IA explicit in our tests since it is enabled by default through
// ENABLE_ALERTING env var.

func TestChannelsAPI(t *testing.T) { //nolint:tparallel
	// TODO Fix this test to run in parallel.
	// t.Parallel()
	client := channelsClient.Default.Channels

	t.Run("add", func(t *testing.T) {
		t.Parallel()

		t.Run("normal", func(t *testing.T) {
			t.Parallel()

			resp, err := client.AddChannel(&channels.AddChannelParams{
				Body: channels.AddChannelBody{
					Summary:  gofakeit.Quote(),
					Disabled: gofakeit.Bool(),
					EmailConfig: &channels.AddChannelParamsBodyEmailConfig{
						SendResolved: false,
						To:           []string{gofakeit.Email()},
					},
				},
				Context: pmmapitests.Context,
			})
			require.NoError(t, err)
			defer deleteChannel(t, client, resp.Payload.ChannelID)

			assert.NotEmpty(t, resp.Payload.ChannelID)
		})

		t.Run("invalid request", func(t *testing.T) {
			t.Parallel()

			resp, err := client.AddChannel(&channels.AddChannelParams{
				Body: channels.AddChannelBody{
					Summary:  gofakeit.Quote(),
					Disabled: gofakeit.Bool(),
					EmailConfig: &channels.AddChannelParamsBodyEmailConfig{
						SendResolved: false,
					},
				},
				Context: pmmapitests.Context,
			})

			pmmapitests.AssertAPIErrorf(t, err, 400, codes.InvalidArgument, "invalid AddChannelRequest.EmailConfig: embedded message failed validation | caused by: invalid EmailConfig.To: value must contain at least 1 item(s)")
			assert.Nil(t, resp)
		})

		t.Run("missing config", func(t *testing.T) {
			t.Parallel()

			resp, err := client.AddChannel(&channels.AddChannelParams{
				Body: channels.AddChannelBody{
					Summary:  gofakeit.Quote(),
					Disabled: gofakeit.Bool(),
				},
				Context: pmmapitests.Context,
			})

			pmmapitests.AssertAPIErrorf(t, err, 400, codes.InvalidArgument, "Missing channel configuration.")
			assert.Nil(t, resp)
		})
	})

	t.Run("change", func(t *testing.T) {
		t.Parallel()

		t.Run("normal", func(t *testing.T) {
			t.Parallel()

			resp1, err := client.AddChannel(&channels.AddChannelParams{
				Body: channels.AddChannelBody{
					Summary:  gofakeit.Quote(),
					Disabled: false,
					EmailConfig: &channels.AddChannelParamsBodyEmailConfig{
						SendResolved: false,
						To:           []string{gofakeit.Email()},
					},
				},
				Context: pmmapitests.Context,
			})
			require.NoError(t, err)
			defer deleteChannel(t, client, resp1.Payload.ChannelID)

			slackChannel := uuid.New().String()
			newSummary := uuid.New().String()
			_, err = client.ChangeChannel(&channels.ChangeChannelParams{
				Body: channels.ChangeChannelBody{
					ChannelID: resp1.Payload.ChannelID,
					Summary:   newSummary,
					Disabled:  true,
					SlackConfig: &channels.ChangeChannelParamsBodySlackConfig{
						SendResolved: true,
						Channel:      slackChannel,
					},
				},
				Context: pmmapitests.Context,
			})
			require.NoError(t, err)

			resp2, err := client.ListChannels(&channels.ListChannelsParams{Context: pmmapitests.Context})
			require.NoError(t, err)

			assert.NotEmpty(t, resp2.Payload.Channels)
			var found bool
			for _, channel := range resp2.Payload.Channels {
				if channel.ChannelID == resp1.Payload.ChannelID {
					assert.Equal(t, newSummary, channel.Summary)
					assert.True(t, channel.Disabled)
					assert.Nil(t, channel.EmailConfig)
					assert.Equal(t, slackChannel, channel.SlackConfig.Channel)
					assert.True(t, channel.SlackConfig.SendResolved)
					found = true
				}
			}

			assert.True(t, found, "Expected channel not found")
		})
	})

	t.Run("remove", func(t *testing.T) {
		t.Parallel()

		t.Run("normal", func(t *testing.T) {
			t.Parallel()

			summary := uuid.New().String()
			resp1, err := client.AddChannel(&channels.AddChannelParams{
				Body: channels.AddChannelBody{
					Summary:  summary,
					Disabled: gofakeit.Bool(),
					EmailConfig: &channels.AddChannelParamsBodyEmailConfig{
						SendResolved: false,
						To:           []string{gofakeit.Email()},
					},
				},
				Context: pmmapitests.Context,
			})
			require.NoError(t, err)

			_, err = client.RemoveChannel(&channels.RemoveChannelParams{
				Body: channels.RemoveChannelBody{
					ChannelID: resp1.Payload.ChannelID,
				},
				Context: pmmapitests.Context,
			})
			require.NoError(t, err)

			resp2, err := client.ListChannels(&channels.ListChannelsParams{Context: pmmapitests.Context})
			require.NoError(t, err)

			for _, channel := range resp2.Payload.Channels {
				assert.NotEqual(t, resp1, channel.ChannelID)
			}
		})

		t.Run("unknown id", func(t *testing.T) {
			t.Parallel()

			resp, err := client.AddChannel(&channels.AddChannelParams{
				Body: channels.AddChannelBody{
					Summary:  gofakeit.Quote(),
					Disabled: gofakeit.Bool(),
					EmailConfig: &channels.AddChannelParamsBodyEmailConfig{
						SendResolved: false,
						To:           []string{gofakeit.Email()},
					},
				},
				Context: pmmapitests.Context,
			})
			require.NoError(t, err)
			defer deleteChannel(t, client, resp.Payload.ChannelID)

			_, err = client.RemoveChannel(&channels.RemoveChannelParams{
				Body: channels.RemoveChannelBody{
					ChannelID: uuid.New().String(),
				},
				Context: pmmapitests.Context,
			})
			require.Error(t, err)
		})

		t.Run("channel in use", func(t *testing.T) {
			t.Parallel()

			templateName := createTemplate(t)
			defer deleteTemplate(t, alertingClient.Default.Alerting, templateName)

			channelID, body := createChannel(t)
			defer deleteChannel(t, channelsClient.Default.Channels, channelID)

			params := createAlertRuleParams(templateName, "", channelID, nil)
			rule, err := channelsClient.Default.Rules.CreateAlertRule(params)
			require.NoError(t, err)
			defer deleteRule(t, channelsClient.Default.Rules, rule.Payload.RuleID)

			_, err = client.RemoveChannel(&channels.RemoveChannelParams{
				Body: channels.RemoveChannelBody{
					ChannelID: channelID,
				},
				Context: pmmapitests.Context,
			})
			pmmapitests.AssertAPIErrorf(t, err, 400, codes.FailedPrecondition, `You can't delete the "%s" channel when it's being used by a rule.`, body.Summary)

			resp, err := client.ListChannels(&channels.ListChannelsParams{
				Context: pmmapitests.Context,
			})
			require.NoError(t, err)

			var found bool
			for _, channel := range resp.Payload.Channels {
				if channelID == channel.ChannelID {
					found = true
				}
			}
			assert.Truef(t, found, "Channel with id %s not found", channelID)
		})
	})

	t.Run("list", func(t *testing.T) {
		client := channelsClient.Default.Channels

		summary := uuid.New().String()
		email := gofakeit.Email()
		disabled := gofakeit.Bool()
		resp1, err := client.AddChannel(&channels.AddChannelParams{
			Body: channels.AddChannelBody{
				Summary:  summary,
				Disabled: disabled,
				EmailConfig: &channels.AddChannelParamsBodyEmailConfig{
					SendResolved: true,
					To:           []string{email},
				},
			},
			Context: pmmapitests.Context,
		})
		require.NoError(t, err)
		defer deleteChannel(t, client, resp1.Payload.ChannelID)

		t.Run("without pagination", func(t *testing.T) {
			resp, err := client.ListChannels(&channels.ListChannelsParams{Context: pmmapitests.Context})
			require.NoError(t, err)

			assert.NotEmpty(t, resp.Payload.Channels)
			var found bool
			for _, channel := range resp.Payload.Channels {
				if channel.ChannelID == resp1.Payload.ChannelID {
					assert.Equal(t, summary, channel.Summary)
					assert.Equal(t, disabled, channel.Disabled)
					assert.Equal(t, []string{email}, channel.EmailConfig.To)
					assert.True(t, channel.EmailConfig.SendResolved)
					found = true
				}
			}
			assert.True(t, found, "Expected channel not found")
		})

		t.Run("pagination", func(t *testing.T) {
			const channelsCount = 5

			channelIds := make(map[string]struct{})

			for i := 0; i < channelsCount; i++ {
				resp, err := client.AddChannel(&channels.AddChannelParams{
					Body: channels.AddChannelBody{
						Summary: gofakeit.Name(),
						EmailConfig: &channels.AddChannelParamsBodyEmailConfig{
							SendResolved: true,
							To:           []string{email},
						},
					},
					Context: pmmapitests.Context,
				})
				require.NoError(t, err)

				channelIds[resp.Payload.ChannelID] = struct{}{}
			}
			defer func() {
				for id := range channelIds {
					deleteChannel(t, client, id)
				}
			}()

			// list channels, so they are all on the first page
			body := channels.ListChannelsBody{
				PageParams: &channels.ListChannelsParamsBodyPageParams{
					PageSize: 20,
					Index:    0,
				},
			}
			listAllChannels, err := client.ListChannels(&channels.ListChannelsParams{
				Body:    body,
				Context: pmmapitests.Context,
			})
			require.NoError(t, err)

			assert.GreaterOrEqual(t, len(listAllChannels.Payload.Channels), channelsCount)
			assert.Equal(t, int32(len(listAllChannels.Payload.Channels)), listAllChannels.Payload.Totals.TotalItems)
			assert.Equal(t, int32(1), listAllChannels.Payload.Totals.TotalPages)

			assertFindChannel := func(list []*channels.ListChannelsOKBodyChannelsItems0, id string) func() bool {
				return func() bool {
					for _, channel := range list {
						if channel.ChannelID == id {
							return true
						}
					}
					return false
				}
			}

			for name := range channelIds {
				assert.Conditionf(t, assertFindChannel(listAllChannels.Payload.Channels, name), "channel %s not found", name)
			}

			// paginate page over page with page size 1 and check the order - it should be the same as in listAllTemplates.
			// last iteration checks that there is no elements for not existing page.
			for pageIndex := 0; pageIndex <= len(listAllChannels.Payload.Channels); pageIndex++ {
				body := channels.ListChannelsBody{
					PageParams: &channels.ListChannelsParamsBodyPageParams{
						PageSize: 1,
						Index:    int32(pageIndex),
					},
				}
				listOneTemplate, err := client.ListChannels(&channels.ListChannelsParams{
					Body: body, Context: pmmapitests.Context,
				})
				require.NoError(t, err)

				assert.Equal(t, listAllChannels.Payload.Totals.TotalItems, listOneTemplate.Payload.Totals.TotalItems)
				assert.GreaterOrEqual(t, listOneTemplate.Payload.Totals.TotalPages, int32(channelsCount))

				if pageIndex != len(listAllChannels.Payload.Channels) {
					require.Len(t, listOneTemplate.Payload.Channels, 1)
					assert.Equal(t, listAllChannels.Payload.Channels[pageIndex].ChannelID, listOneTemplate.Payload.Channels[0].ChannelID)
				} else {
					assert.Len(t, listOneTemplate.Payload.Channels, 0)
				}
			}
		})
	})
}

func deleteChannel(t *testing.T, client channels.ClientService, id string) {
	t.Helper()
	_, err := client.RemoveChannel(&channels.RemoveChannelParams{
		Body: channels.RemoveChannelBody{
			ChannelID: id,
		},
		Context: pmmapitests.Context,
	})
	assert.NoError(t, err)
}
