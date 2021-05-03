// pmm-admin
// Copyright 2019 Percona LLC
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

// Package helpers provides helpers for whole pmm-admin.
package helpers

import (
	"fmt"

	"github.com/percona/pmm/version"

	"github.com/percona/pmm-admin/agentlocal"
)

// HAProxyMinPMMServerVersion contains minimum version for running HAProxy.
const haProxyMinPMMServerVersion = "2.15.0"

// ServerVersionLessThan return if provided version is lower than server version.
func ServerVersionLessThan(currentVersion string) (bool, error) {
	status, err := agentlocal.GetStatus(agentlocal.DoNotRequestNetworkInfo)
	if err != nil {
		return false, err
	}

	v, err := version.Parse(status.ServerVersion)
	if err != nil {
		return false, err
	}
	v.Rest = ""

	// Check if version meets the conditions
	minVersion, err := version.Parse(currentVersion)
	if err != nil {
		return false, err
	}

	return v.Less(minVersion), nil
}

// IsHAProxySupported return if is HAProxy supported on running PMM server.
func IsHAProxySupported() (bool, error) {
	minVersion := haProxyMinPMMServerVersion
	lessThanMinVersion, err := ServerVersionLessThan(minVersion)
	if err != nil {
		return false, err
	}
	if lessThanMinVersion {
		return false, fmt.Errorf("haproxy is not supported in this version, please update your pmm-server to %s or higher", minVersion)
	}

	return true, nil
}
