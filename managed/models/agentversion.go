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

package models

import (
	"fmt"

	"github.com/hashicorp/go-version"
	"github.com/pkg/errors"
	"gopkg.in/reform.v1"
)

// PMMAgentMinVersionForPostgreSQLSSLSni is the minimum required version of PMM Agent.
var PMMAgentMinVersionForPostgreSQLSSLSni = version.Must(version.NewVersion("2.41.0-0"))

// AgentNotSupportedError is used when the target PMM agent doesn't support the requested functionality.
type AgentNotSupportedError struct {
	Functionality   string
	AgentID         string
	AgentVersion    string
	MinAgentVersion string
}

func (e AgentNotSupportedError) Error() string {
	return fmt.Sprintf("'%s' functionality is not supported by pmm-agent %q version %q. Required minimum version is %q", e.Functionality,
		e.AgentID, e.AgentVersion, e.MinAgentVersion)
}

// PMMAgentSupported checks if pmm agent version satisfies required min version.
func PMMAgentSupported(q *reform.Querier, pmmAgentID, functionalityPrefix string, pmmMinVersion *version.Version) error {
	pmmAgent, err := FindAgentByID(q, pmmAgentID)
	if err != nil {
		return errors.Errorf("failed to get PMM Agent: %s", err)
	}
	return IsAgentSupported(pmmAgent, functionalityPrefix, pmmMinVersion)
}

// IsAgentSupported contains logic for PMMAgentSupported.
func IsAgentSupported(agentModel *Agent, functionalityPrefix string, pmmMinVersion *version.Version) error {
	if agentModel == nil {
		return errors.New("nil agent")
	}
	if agentModel.Version == nil {
		return errors.Errorf("pmm agent %q has no version info", agentModel.AgentID)
	}
	pmmAgentVersion, err := version.NewVersion(*agentModel.Version)
	if err != nil {
		return errors.Errorf("failed to parse PMM agent version %q: %s", *agentModel.Version, err)
	}

	if pmmAgentVersion.LessThan(pmmMinVersion) {
		return errors.WithStack(AgentNotSupportedError{
			AgentID:         agentModel.AgentID,
			Functionality:   functionalityPrefix,
			AgentVersion:    *agentModel.Version,
			MinAgentVersion: pmmMinVersion.String(),
		})
	}
	return nil
}

// IsPostgreSQLSSLSniSupported checks if PostgreSQL SSL SNI is supported by the PMM Agent.
func IsPostgreSQLSSLSniSupported(q *reform.Querier, pmmAgentID string) (bool, error) {
	err := PMMAgentSupported(q, pmmAgentID, "postgresql SSL sni check", PMMAgentMinVersionForPostgreSQLSSLSni)
	switch {
	case errors.As(err, &AgentNotSupportedError{}):
		return false, nil
	case err == nil:
		return true, nil
	default:
		return false, errors.Wrap(err, "couldn't compare PMM Agent version")
	}
}
