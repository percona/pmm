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

// HAParams defines parameters related to High Availability.
type HAParams struct {
	GrafanaGossipPort int
	// Enabled indicates whether HA is enabled.
	Enabled          bool
	NodeID           string
	AdvertiseAddress string
	// Nodes is a list of initial cluster node addresses.
	Nodes      []string
	RaftPort   int
	GossipPort int
}

// Params defines parameters for supervisor.
type Params struct {
	HAParams *HAParams
	VMParams *VictoriaMetricsParams
	PGParams *PGParams
}
