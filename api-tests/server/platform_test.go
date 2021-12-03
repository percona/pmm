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

package server

import (
	"net/http"
	"os"
	"os/user"
	"strings"
	"testing"

	"github.com/brianvoe/gofakeit"
	platformClient "github.com/percona/pmm/api/platformpb/json/client"
	"github.com/percona/pmm/api/platformpb/json/client/platform"
	serverClient "github.com/percona/pmm/api/serverpb/json/client"
	"github.com/percona/pmm/api/serverpb/json/client/server"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"

	pmmapitests "github.com/percona/pmm-managed/api-tests"
)

func TestPlatform(t *testing.T) {
	client := platformClient.Default.Platform
	serverClient := serverClient.Default.Server
	t.Run("connect", func(t *testing.T) {
		const serverName string = "my PMM"
		email, password := os.Getenv("PERCONA_TEST_PORTAL_EMAIL"), os.Getenv("PERCONA_TEST_PORTAL_PASSWORD")
		if email == "" || password == "" {
			t.Skip("Environment variables PERCONA_TEST_PORTAL_EMAIL, PERCONA_TEST_PORTAL_PASSWORD not set.")
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
					Email:      email,
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
					Email:      email,
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
					Email:      email,
					Password:   password,
					ServerName: "",
				},
				Context: pmmapitests.Context,
			})
			pmmapitests.AssertAPIErrorf(t, err, http.StatusBadRequest, codes.InvalidArgument, "invalid field ServerName: value '' must not be an empty string")
		})

		t.Run("successful call", func(t *testing.T) {
			t.Skip("Skip this test until we've got disconnect")

			_, err := client.Connect(&platform.ConnectParams{
				Body: platform.ConnectBody{
					ServerName: serverName,
					Email:      email,
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
		})
	})
}

// genCredentials creates test user email, password, firstName and lastName.
func genCredentials(t *testing.T) (string, string, string, string) {
	hostname, err := os.Hostname()
	require.NoError(t, err)

	u, err := user.Current()
	require.NoError(t, err)

	email := strings.Join([]string{u.Username, hostname, gofakeit.Email(), "test"}, ".")
	password := gofakeit.Password(true, true, true, false, false, 14)
	firstName := gofakeit.FirstName()
	lastName := gofakeit.LastName()
	return email, password, firstName, lastName
}
