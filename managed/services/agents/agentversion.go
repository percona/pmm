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

package agents

import (
	"github.com/hashicorp/go-version"
	"github.com/pkg/errors"
	"gopkg.in/reform.v1"

	"github.com/percona/pmm/managed/models"
)

var (
	// ErrIncompatibleAgentVersion is returned when the agent version is incompatible for making a backup or restore.
	ErrIncompatibleAgentVersion = errors.New("incompatible agent version")
)

// PMMAgentSupported checks if pmm agent version satisfies required min version.
func PMMAgentSupported(q *reform.Querier, pmmAgentID, functionalityPrefix string, pmmMinVersion *version.Version) error {
	pmmAgent, err := models.FindAgentByID(q, pmmAgentID)
	if err != nil {
		return errors.Errorf("failed to get PMM Agent: %s", err)
	}
	if pmmAgent.Version == nil {
		return errors.Errorf("pmm agent %q has no version info", pmmAgentID)
	}
	pmmAgentVersion, err := version.NewVersion(*pmmAgent.Version)
	if err != nil {
		return errors.Errorf("failed to parse PMM agent version %q: %s", *pmmAgent.Version, err)
	}

	if pmmAgentVersion.LessThan(pmmMinVersion) {
		return errors.WithMessagef(ErrIncompatibleAgentVersion, "%s is not supported on pmm-agent %q version %q, min required version %q", functionalityPrefix,
			pmmAgentID, *pmmAgent.Version, pmmMinVersion)
	}

	return nil
}
