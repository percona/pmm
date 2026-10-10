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

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/percona/pmm/version"
)

// RTAMinAgentVersion returns the first pmm-agent release that ships the Real-Time Analytics
// collector for the given service type, and whether RTA is supported for that type at all.
// Collectors for different databases landed in different releases, so the gate is per service type.
func RTAMinAgentVersion(serviceType ServiceType) (version.FeatureVersion, bool) {
	switch serviceType {
	case MongoDBServiceType:
		return version.MongoDBRtaAgentSupportVersion, true
	case MySQLServiceType:
		return version.MySQLRtaAgentSupportVersion, true
	default:
		return nil, false
	}
}

// IsRTASupported reports whether a pmm-agent of the given version can run Real-Time Analytics
// for the given service type. Unparsable versions and service types without RTA are unsupported.
func IsRTASupported(pmmAgentVersion string, serviceType ServiceType) bool {
	minVersion, ok := RTAMinAgentVersion(serviceType)
	if !ok {
		return false
	}

	parsed, err := version.Parse(pmmAgentVersion)
	if err != nil {
		return false
	}

	return parsed.IsFeatureSupported(minVersion)
}

// RTANotSupportedMessage explains why a service's pmm-agent cannot run Real-Time Analytics,
// naming the pmm-agent version required for the service type. The service is named as well as
// identified: the ID alone leaves the reader to look up which service was refused.
func RTANotSupportedMessage(serviceName, serviceID, pmmAgentVersion string, serviceType ServiceType) string {
	if pmmAgentVersion == "" {
		pmmAgentVersion = "unknown"
	}

	minVersion, ok := RTAMinAgentVersion(serviceType)
	if !ok {
		return fmt.Sprintf("Service %s (id %s) of type %s does not support Real-Time Analytics.", serviceName, serviceID, serviceType)
	}

	return fmt.Sprintf("Service %s (id %s) has pmm-agent with version %s not supporting Real-Time Analytics; "+
		"pmm-agent %d.%d.%d or later is required.", serviceName, serviceID, pmmAgentVersion, minVersion.Major, minVersion.Minor, minVersion.Patch)
}

// RTANotSupportedError is the FailedPrecondition error for RTANotSupportedMessage.
func RTANotSupportedError(serviceName, serviceID, pmmAgentVersion string, serviceType ServiceType) error {
	return status.Error(codes.FailedPrecondition, RTANotSupportedMessage(serviceName, serviceID, pmmAgentVersion, serviceType))
}
