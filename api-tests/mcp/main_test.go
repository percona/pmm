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
	"testing"

	"github.com/sirupsen/logrus"

	pmmapitests "github.com/percona/pmm/api-tests"
	serverClient "github.com/percona/pmm/api/server/v1/json/client"
	"github.com/percona/pmm/api/server/v1/json/client/server_service"
)

// TestMain enables the MCP endpoint, which is off by default, for the package
// and restores it afterwards; a server that refuses the change fails the run.
func TestMain(m *testing.M) {
	l := logrus.WithField("component", "mcp-api-tests")
	res, err := serverClient.Default.ServerService.GetSettings(&server_service.GetSettingsParams{Context: pmmapitests.Context})
	if err != nil {
		l.Fatalf("Cannot read PMM Server settings: %s.", err)
	}

	if !res.Payload.Settings.EnableMcp {
		err = setMCPEnabled(true)
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
