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

// Package env provides simple environment variable utilities without dependencies on models.
package env

import (
	"os"
	"strconv"
)

const (
	// PlatformInsecure allows PMM to skip TLS verification when connecting to Percona Platform.
	PlatformInsecure = "PMM_DEV_PERCONA_PLATFORM_INSECURE"

	// PlatformPublicKey is used to store the public key for Percona Platform.
	PlatformPublicKey = "PMM_DEV_PERCONA_PLATFORM_PUBLIC_KEY"

	// InterfaceToBind specifies the network interface that the PMM Server should bind to.
	InterfaceToBind = "PMM_INTERFACE_TO_BIND"

	// EnableAccessControl is used to enable Access Control in PMM.
	EnableAccessControl = "PMM_ENABLE_ACCESS_CONTROL"

	// PlatformAPITimeout specifies the timeout for Percona Platform API requests.
	PlatformAPITimeout = "PMM_DEV_PERCONA_PLATFORM_API_TIMEOUT"

	// PlatformAddress is the environment variable name used to store the URL for Percona Platform.
	PlatformAddress = "PMM_DEV_PERCONA_PLATFORM_ADDRESS"

	// EnableInternalPgQAN is used to enable Query Analytics for PMM's internal PostgreSQL.
	EnableInternalPgQAN = "PMM_ENABLE_INTERNAL_PG_QAN"
)

// GetBool returns the boolean value of the environment variable.
// Returns false if the variable is not set or cannot be parsed as boolean.
// It does not return errors since it assumes that validation has already been done during startup.
func GetBool(key string) bool {
	v, ok := os.LookupEnv(key)
	if !ok {
		return false
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		return false
	}
	return b
}
