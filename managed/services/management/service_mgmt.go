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

package management

import (
	"gopkg.in/reform.v1"

	managementv1 "github.com/percona/pmm/api/management/v1/service"
)

// MgmtServiceService is a management service for working with services.
type MgmtServiceService struct {
	db       *reform.DB
	r        agentsRegistry
	state    agentsStateUpdater
	vmdb     prometheusService
	vmClient victoriaMetricsClient

	managementv1.UnimplementedManagementV1Beta1ServiceServer
}

type statusMetrics struct {
	status      int
	serviceType string
}

// NewMgmtServiceService creates MgmtServiceService instance.
func NewMgmtServiceService(db *reform.DB, r agentsRegistry, state agentsStateUpdater, vmdb prometheusService, vmClient victoriaMetricsClient) *MgmtServiceService {
	return &MgmtServiceService{
		db:       db,
		r:        r,
		state:    state,
		vmdb:     vmdb,
		vmClient: vmClient,
	}
}
