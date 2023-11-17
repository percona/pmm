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
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"

	pmmapitests "github.com/percona/pmm/api-tests"
	platformClient "github.com/percona/pmm/api/platformpb/v1/json/client"
	platform "github.com/percona/pmm/api/platformpb/v1/json/client/platform_service"
	serverClient "github.com/percona/pmm/api/serverpb/json/client"
	server "github.com/percona/pmm/api/serverpb/json/client/server_service"
)

func TestPlatform(t *testing.T) {
	client := platformClient.Default.PlatformService
	serverClient := serverClient.Default.ServerService

	const serverName = string("my PMM")
	username, password := os.Getenv("PERCONA_TEST_PORTAL_USERNAME"), os.Getenv("PERCONA_TEST_PORTAL_PASSWORD")
	t.Run("connect and disconnect", func(t *testing.T) {
		if username == "" || password == "" {
			t.Skip("Environment variables PERCONA_TEST_PORTAL_USERNAME, PERCONA_TEST_PORTAL_PASSWORD not set.")
		}
		t.Run("PMM server does not have address set", func(t *testing.T) {
			_, err := client.Connect(&platform.ConnectParams{
				Body: platform.ConnectBody{
					Email:      "wrong@example.com",
					Password:   password,
					ServerName: serverName,
				},
				Context: pmmapitests.Context,
			})
			pmmapitests.AssertAPIErrorf(t, err, http.StatusBadRequest, codes.FailedPrecondition, "The address of PMM server is not set")
		})

		// Set the PMM address to localhost.
		res, err := serverClient.ChangeSettings(&server.ChangeSettingsParams{
			Body: server.ChangeSettingsBody{
				PMMPublicAddress: "localhost",
			},
			Context: pmmapitests.Context,
		})
		require.NoError(t, err)
		assert.Equal(t, "localhost", res.Payload.Settings.PMMPublicAddress)

		t.Run("wrong email", func(t *testing.T) {
			_, err := client.Connect(&platform.ConnectParams{
				Body: platform.ConnectBody{
					Email:      "wrong@example.com",
					Password:   password,
					ServerName: serverName,
				},
				Context: pmmapitests.Context,
			})
			pmmapitests.AssertAPIErrorf(t, err, http.StatusUnauthorized, codes.Unauthenticated, "Incorrect username or password.")
		})

		t.Run("wrong password", func(t *testing.T) {
			_, err := client.Connect(&platform.ConnectParams{
				Body: platform.ConnectBody{
					Email:      username,
					Password:   "WrongPassword12345",
					ServerName: serverName,
				},
				Context: pmmapitests.Context,
			})
			pmmapitests.AssertAPIErrorf(t, err, http.StatusUnauthorized, codes.Unauthenticated, "Incorrect username or password.")
		})

		t.Run("empty email", func(t *testing.T) {
			_, err := client.Connect(&platform.ConnectParams{
				Body: platform.ConnectBody{
					Email:      "",
					Password:   password,
					ServerName: serverName,
				},
				Context: pmmapitests.Context,
			})
			pmmapitests.AssertAPIErrorf(t, err, http.StatusBadRequest, codes.InvalidArgument, "invalid field Email: value '' must not be an empty string")
		})

		t.Run("empty password", func(t *testing.T) {
			_, err := client.Connect(&platform.ConnectParams{
				Body: platform.ConnectBody{
					Email:      username,
					ServerName: serverName,
					Password:   "",
				},
				Context: pmmapitests.Context,
			})
			pmmapitests.AssertAPIErrorf(t, err, http.StatusBadRequest, codes.InvalidArgument, "invalid field Password: value '' must not be an empty string")
		})

		t.Run("empty server name", func(t *testing.T) {
			_, err := client.Connect(&platform.ConnectParams{
				Body: platform.ConnectBody{
					Email:      username,
					Password:   password,
					ServerName: "",
				},
				Context: pmmapitests.Context,
			})
			pmmapitests.AssertAPIErrorf(t, err, http.StatusBadRequest, codes.InvalidArgument, "invalid field ServerName: value '' must not be an empty string")
		})

		t.Run("successful connect and disconnect", func(t *testing.T) {
			_, err := client.Connect(&platform.ConnectParams{
				Body: platform.ConnectBody{
					ServerName: serverName,
					Email:      username,
					Password:   password,
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
		if username == "" || password == "" {
			t.Skip("Environment variables PERCONA_TEST_PORTAL_USERNAME, PERCONA_TEST_PORTAL_PASSWORD not set.")
		}

		t.Run("success", func(t *testing.T) {
			_, err := client.Connect(&platform.ConnectParams{
				Body: platform.ConnectBody{
					ServerName: serverName,
					Email:      username,
					Password:   password,
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
		if username == "" || password == "" {
			t.Skip("Environment variables PERCONA_TEST_PORTAL_USERNAME, PERCONA_TEST_PORTAL_PASSWORD not set.")
		}

		t.Run("success", func(t *testing.T) {
			_, err := client.Connect(&platform.ConnectParams{
				Body: platform.ConnectBody{
					ServerName: serverName,
					Email:      username,
					Password:   password,
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
		if username == "" || password == "" {
			t.Skip("Environment variables PERCONA_TEST_PORTAL_USERNAME, PERCONA_TEST_PORTAL_PASSWORD not set.")
		}

		t.Run("success", func(t *testing.T) {
			_, err := client.Connect(&platform.ConnectParams{
				Body: platform.ConnectBody{
					ServerName: serverName,
					Email:      username,
					Password:   password,
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
