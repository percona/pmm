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

package mcp

import (
	"errors"
	"testing"

	"github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"

	pmmapitests "github.com/percona/pmm/api-tests"
	serverClient "github.com/percona/pmm/api/server/v1/json/client"
	"github.com/percona/pmm/api/server/v1/json/client/server_service"
)

// TestMain switches the MCP endpoint on for the duration of the package and
// restores the previous value afterwards.
//
// The endpoint is off by default (PMM_ENABLE_MCP=false), and these tests run
// against whatever server they are pointed at - a Feature Build in CI, a dev
// environment locally - so they must not depend on how that server was
// started. Settings are read per request, so the change applies at once. A
// server started with PMM_ENABLE_MCP=false refuses the change, and the package
// is skipped.
func TestMain(m *testing.M) {
	l := logrus.WithField("component", "mcp-api-tests")
	res, err := serverClient.Default.ServerService.GetSettings(&server_service.GetSettingsParams{Context: pmmapitests.Context})
	if err != nil {
		l.Fatalf("Cannot read PMM Server settings: %s.", err)
	}

	if !res.Payload.Settings.EnableMcp {
		err = setMCPEnabled(true)
		var refused *server_service.ChangeSettingsDefault
		if errors.As(err, &refused) && refused.Payload != nil && refused.Payload.Code == int32(codes.FailedPrecondition) {
			l.Warnf("Skipping the MCP API tests: the endpoint cannot be enabled (%s).", err)
			return
		}
		if err != nil {
			l.Fatalf("Cannot enable the MCP endpoint: %s.", err)
		}
		defer func() {
			err := setMCPEnabled(false)
			if err != nil {
				l.Warnf("Cannot restore the MCP endpoint to disabled: %s.", err)
			}
		}()
	}

	m.Run()
}

func setMCPEnabled(enable bool) error {
	_, err := serverClient.Default.ServerService.ChangeSettings(&server_service.ChangeSettingsParams{
		Body:    server_service.ChangeSettingsBody{EnableMcp: new(enable)},
		Context: pmmapitests.Context,
	})
	return err
}
