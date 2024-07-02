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
	"net/http"
	"testing"

	"github.com/AlekSi/pointer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"

	pmmapitests "github.com/percona/pmm/api-tests"
	platformClient "github.com/percona/pmm/api/platform/v1/json/client"
	platform "github.com/percona/pmm/api/platform/v1/json/client/platform_service"
	serverClient "github.com/percona/pmm/api/server/v1/json/client"
	server "github.com/percona/pmm/api/server/v1/json/client/server_service"
)

func TestPlatform(t *testing.T) {
	client := platformClient.Default.PlatformService
	serverClient := serverClient.Default.ServerService

	// TODO: provide a real PersonalAccessToken for the dev environment so this test can be run.
	// The one below is a fake, it is there to fix the test compilation.

	const serverName = string("my PMM")
	t.Run("connect and disconnect", func(t *testing.T) {
		t.Run("PMM server does not have address set", func(t *testing.T) {
			_, err := client.Connect(&platform.ConnectParams{
				Body: platform.ConnectBody{
					PersonalAccessToken: "JReeZA5EqM4b6bZMxBOEaAxoc4rWd5teK7HF",
					ServerName:          serverName,
				},
				Context: pmmapitests.Context,
			})
			pmmapitests.AssertAPIErrorf(t, err, http.StatusBadRequest, codes.FailedPrecondition, "The address of PMM server is not set")
		})

		// Set the PMM address to localhost.
		res, err := serverClient.ChangeSettings(&server.ChangeSettingsParams{
			Body: server.ChangeSettingsBody{
				PMMPublicAddress: pointer.ToString("localhost"),
			},
			Context: pmmapitests.Context,
		})
		require.NoError(t, err)
		assert.Equal(t, "localhost", res.Payload.Settings.PMMPublicAddress)

		t.Run("empty access token", func(t *testing.T) {
			_, err := client.Connect(&platform.ConnectParams{
				Body: platform.ConnectBody{
					PersonalAccessToken: "",
					ServerName:          serverName,
				},
				Context: pmmapitests.Context,
			})
			pmmapitests.AssertAPIErrorf(t, err, http.StatusBadRequest, codes.InvalidArgument, "invalid field PersonalAccessToken: value '' must not be an empty string")
		})

		t.Run("empty server name", func(t *testing.T) {
			_, err := client.Connect(&platform.ConnectParams{
				Body: platform.ConnectBody{
					PersonalAccessToken: "JReeZA5EqM4b6bZMxBOEaAxoc4rWd5teK7HF",
					ServerName:          "",
				},
				Context: pmmapitests.Context,
			})
			pmmapitests.AssertAPIErrorf(t, err, http.StatusBadRequest, codes.InvalidArgument, "invalid field ConnectRequest.ServerName: value '' must not be an empty string")
		})

		t.Run("successful connect and disconnect", func(t *testing.T) {
			_, err := client.Connect(&platform.ConnectParams{
				Body: platform.ConnectBody{
					ServerName:          serverName,
					PersonalAccessToken: "JReeZA5EqM4b6bZMxBOEaAxoc4rWd5teK7HF",
				},
				Context: pmmapitests.Context,
			})
			require.NoError(t, err)

			// Confirm we are connected to Portal.
			settings, err := serverClient.GetSettings(nil)
			require.NoError(t, err)
			require.NotNil(t, settings)
			assert.True(t, settings.GetPayload().Settings.ConnectedToPlatform)

			_, err = client.Disconnect(&platform.DisconnectParams{
				Context: pmmapitests.Context,
			})
			require.NoError(t, err)

			// Confirm we are disconnected from Portal.
			settings, err = serverClient.GetSettings(nil)
			require.NoError(t, err)
			require.NotNil(t, settings)
			assert.False(t, settings.GetPayload().Settings.ConnectedToPlatform)
		})
	})

	t.Run("search tickets", func(t *testing.T) {
		t.Run("success", func(t *testing.T) {
			_, err := client.Connect(&platform.ConnectParams{
				Body: platform.ConnectBody{
					ServerName:          serverName,
					PersonalAccessToken: "JReeZA5EqM4b6bZMxBOEaAxoc4rWd5teK7HF",
				},
				Context: pmmapitests.Context,
			})
			require.NoError(t, err)

			// Confirm we are connected to Portal.
			settings, err := serverClient.GetSettings(nil)
			require.NoError(t, err)
			require.NotNil(t, settings)
			assert.True(t, settings.GetPayload().Settings.ConnectedToPlatform)

			resp, err := client.SearchOrganizationTickets(&platform.SearchOrganizationTicketsParams{Context: pmmapitests.Context})
			require.NoError(t, err)
			require.NotNil(t, resp.GetPayload().Tickets)
		})
	})

	t.Run("search entitlements", func(t *testing.T) { //nolint:dupl
		t.Run("success", func(t *testing.T) {
			_, err := client.Connect(&platform.ConnectParams{
				Body: platform.ConnectBody{
					ServerName:          serverName,
					PersonalAccessToken: "JReeZA5EqM4b6bZMxBOEaAxoc4rWd5teK7HF",
				},
				Context: pmmapitests.Context,
			})
			require.NoError(t, err)

			// Confirm we are connected to Portal.
			settings, err := serverClient.GetSettings(nil)
			require.NoError(t, err)
			require.NotNil(t, settings)
			assert.True(t, settings.GetPayload().Settings.ConnectedToPlatform)

			resp, err := client.SearchOrganizationEntitlements(&platform.SearchOrganizationEntitlementsParams{Context: pmmapitests.Context})
			require.NoError(t, err)
			require.NotNil(t, resp.GetPayload().Entitlements)
		})
	})

	t.Run("get contact information", func(t *testing.T) { //nolint:dupl
		t.Run("success", func(t *testing.T) {
			_, err := client.Connect(&platform.ConnectParams{
				Body: platform.ConnectBody{
					ServerName:          serverName,
					PersonalAccessToken: "JReeZA5EqM4b6bZMxBOEaAxoc4rWd5teK7HF",
				},
				Context: pmmapitests.Context,
			})
			require.NoError(t, err)

			// Confirm we are connected to Portal.
			settings, err := serverClient.GetSettings(nil)
			require.NoError(t, err)
			require.NotNil(t, settings)
			assert.True(t, settings.GetPayload().Settings.ConnectedToPlatform)

			resp, err := client.GetContactInformation(&platform.GetContactInformationParams{Context: pmmapitests.Context})
			require.NoError(t, err)
			require.NotEmpty(t, resp.GetPayload().CustomerSuccess.Email)
			require.NotEmpty(t, resp.GetPayload().CustomerSuccess.Name)
			require.NotEmpty(t, resp.GetPayload().NewTicketURL)
		})
	})
}
