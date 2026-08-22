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

package common

import (
	"strings"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/percona/pmm/managed/models"
)

// mongoDBExporterReservedEnvVars are environment variable names pmm-agent's supervisor always
// sets itself for mongodb_exporter (see mongodbExporterConfig in managed/services/agents/mongodb.go).
// A user-selected name here would never take effect: the supervisor skips it to avoid overriding
// the computed value, silently, on the agent side. Rejecting it here instead gives the caller an
// actionable error at request time.
var mongoDBExporterReservedEnvVars = map[string]struct{}{
	"MONGODB_URI": {},
}

// ValidateMongoDBExporterEnvVarNames rejects environment variable names that pmm-agent reserves
// for mongodb_exporter itself, except for names already present in grandfathered: this field is
// full-replace, so an exporter that already stores a reserved name (e.g. from before this check
// existed) must still be able to resend it while changing other, unrelated names. Pass a nil
// grandfathered when there is no existing agent to grandfather, such as on Add.
// Grandfathered matches case-sensitively (unlike the reserved-name check below, which is
// case-insensitive on principle): a stored "mongodb_uri" is a different OS environment variable
// from "MONGODB_URI" and must not grandfather a switch to the latter.
// It lives in this cross-service package (rather than services/inventory, where the check
// originated) so both the inventory API and ManagementService.addMongoDB can apply the same check
// without either service importing the other.
func ValidateMongoDBExporterEnvVarNames(names []string, grandfathered map[string]struct{}) error {
	for _, name := range names {
		trimmed := strings.TrimSpace(name)
		if _, ok := grandfathered[trimmed]; ok {
			continue
		}
		if _, ok := mongoDBExporterReservedEnvVars[strings.ToUpper(trimmed)]; ok {
			return status.Errorf(codes.InvalidArgument,
				"environment variable name %q is set by pmm-agent for mongodb_exporter and cannot be selected", name)
		}
	}

	return nil
}

// MongoDBExporterEnvVarNamesGrandfathered returns agent's currently-stored environment variable
// names, normalized for ValidateMongoDBExporterEnvVarNames's grandfathered set. Agent must be the
// row the caller intends to update, read within the same transaction as that update, so the names
// it grandfathers cannot go stale before the update actually applies them.
func MongoDBExporterEnvVarNamesGrandfathered(agent *models.Agent) (map[string]struct{}, error) {
	existing, err := agent.GetEnvironmentVariableNames()
	if err != nil {
		return nil, err
	}

	grandfathered := make(map[string]struct{}, len(existing))
	for _, name := range existing {
		grandfathered[strings.TrimSpace(name)] = struct{}{}
	}

	return grandfathered, nil
}
