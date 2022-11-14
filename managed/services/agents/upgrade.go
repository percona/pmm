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
	"fmt"

	"github.com/pkg/errors"

	"github.com/percona/pmm/api/agentpb"
)

// Upgrader handles upgrade/downgrade of agent version.
type Upgrader struct {
	r *Registry
}

// NewUpgrader creates new upgrader instance.
func NewUpgrader(r *Registry) *Upgrader {
	return &Upgrader{
		r: r,
	}
}

var (
	ErrInvalidResponse = errors.New("InvalidResponse")
	ErrUpgradeError    = errors.New("UpgradeError")
)

// RequestUpgrade requests upgrade of PMM Agent to the same version as PMM Server.
func (u *Upgrader) RequestUpgrade(pmmAgentID string) error {
	agent, err := u.r.get(pmmAgentID)
	if err != nil {
		return err
	}

	res, err := agent.channel.SendAndWaitResponse(&agentpb.StartUpdateRequest{
		Version: "2.30.0",
	})
	if err != nil {
		return err
	}

	updateRes, ok := res.(*agentpb.StartUpdateResponse)
	if !ok {
		return fmt.Errorf("%w: received invalid response from agent", ErrInvalidResponse)
	}

	if updateRes.Error != "" {
		return fmt.Errorf("%w: "+updateRes.Error, ErrUpgradeError)
	}

	return nil
}
