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

package services

import (
	"maps"

	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/managed/pi/check"
)

// Target contains required info about advisor check target.
type Target struct {
	AgentID        string
	ServiceID      string
	ServiceName    string
	ServiceType    models.ServiceType
	NodeID         string
	NodeName       string
	Environment    string
	Cluster        string
	ReplicationSet string
	Region         string
	AZ             string
	Labels         map[string]string
	DSN            string
	Files          map[string]string
	TDP            *models.DelimiterPair
	TLSSkipVerify  bool
}

// Copy creates a copy of the Target instance.
func (t *Target) Copy() Target {
	labels := make(map[string]string, len(t.Labels))
	maps.Copy(labels, t.Labels)

	files := make(map[string]string, len(t.Files))
	maps.Copy(files, t.Files)

	return Target{
		AgentID:        t.AgentID,
		ServiceID:      t.ServiceID,
		ServiceName:    t.ServiceName,
		ServiceType:    t.ServiceType,
		NodeID:         t.NodeID,
		NodeName:       t.NodeName,
		Environment:    t.Environment,
		Cluster:        t.Cluster,
		ReplicationSet: t.ReplicationSet,
		Region:         t.Region,
		AZ:             t.AZ,
		Labels:         labels,
		DSN:            t.DSN,
		Files:          files,
		TDP:            new(*t.TDP),
		TLSSkipVerify:  t.TLSSkipVerify,
	}
}

// CheckResult contains the output from the check file and other information.
type CheckResult struct {
	CheckName string
	Category  string
	Interval  check.Interval
	Target    Target
	Result    check.Result
}
