// Copyright (C) 2023 Percona LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//  http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package client contains business logic of working with pmm-managed.
package client

import (
	"github.com/pkg/errors"

	"github.com/percona/pmm/agent/versioner"
	agentv1 "github.com/percona/pmm/api/agent/v1"
)

func (c *Client) handleVersionsRequest(r *agentv1.GetVersionsRequest) []*agentv1.GetVersionsResponse_Version {
	versionsResponse := make([]*agentv1.GetVersionsResponse_Version, 0, len(r.Softwares))
	for _, s := range r.Softwares {
		var version string
		var err error
		switch s.Software.(type) {
		case *agentv1.GetVersionsRequest_Software_Mysqld:
			version, err = c.softwareVersioner.MySQLdVersion()
		case *agentv1.GetVersionsRequest_Software_Xtrabackup:
			version, err = c.softwareVersioner.XtrabackupVersion()
		case *agentv1.GetVersionsRequest_Software_Xbcloud:
			version, err = c.softwareVersioner.XbcloudVersion()
		case *agentv1.GetVersionsRequest_Software_Qpress:
			version, err = c.softwareVersioner.QpressVersion()
		case *agentv1.GetVersionsRequest_Software_Mongod:
			version, err = c.softwareVersioner.MongoDBVersion()
		case *agentv1.GetVersionsRequest_Software_Pbm:
			version, err = c.softwareVersioner.PBMVersion()
		default:
			err = errors.Errorf("unknown software type %T", s.Software)
		}

		if err != nil && !errors.Is(err, versioner.ErrNotFound) {
			versionsResponse = append(versionsResponse, &agentv1.GetVersionsResponse_Version{Error: err.Error()})
			continue
		}

		versionsResponse = append(versionsResponse, &agentv1.GetVersionsResponse_Version{Version: version})
	}

	return versionsResponse
}
