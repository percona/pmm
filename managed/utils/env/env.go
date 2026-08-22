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
	"fmt"
	"os"
	"strconv"
	"strings"
)

const (
	// PlatformInsecure allows PMM to skip TLS verification when connecting to Percona Platform.
	PlatformInsecure = "PMM_DEV_PERCONA_PLATFORM_INSECURE"

	// InterfaceToBind specifies the network interface that the PMM Server should bind to.
	InterfaceToBind = "PMM_INTERFACE_TO_BIND"

	// EnableAccessControl is used to enable Access Control in PMM.
	EnableAccessControl = "PMM_ENABLE_ACCESS_CONTROL"

	// PlatformAPITimeout specifies the timeout for Percona Platform API requests.
	PlatformAPITimeout = "PMM_DEV_PERCONA_PLATFORM_API_TIMEOUT"

	// PlatformAddress is the environment variable name used to store the URL for Percona Platform.
	PlatformAddress = "PMM_PERCONA_PLATFORM_ADDRESS"

	// EnableInternalPgQAN is used to enable Query Analytics for PMM's internal PostgreSQL.
	EnableInternalPgQAN = "PMM_ENABLE_INTERNAL_PG_QAN"

	// ClickHouseNodes is used to store the ClickHouse nodes.
	ClickHouseNodes = "PMM_CLICKHOUSE_NODES"

	// ClickHouseConfig specifies the configuration for ClickHouse.
	ClickHouseConfig = "PMM_CLICKHOUSE_CONFIG"
)

// LookupBool returns the boolean value of the environment variable. It tells the three states
// of an optional boolean apart:
//   - (nil, nil) when the variable is not set,
//   - (value, nil) when it holds a boolean,
//   - (nil, error) when it is set to something that is not a boolean.
//
// The error lets a caller decide what an unparsable value means for it, instead of having that
// choice made here. The envvars.ParseEnvVars parser reports the same values as configuration
// errors, and pmm-managed-init refuses to start PMM Server when it does.
func LookupBool(key string) (*bool, error) {
	v, ok := os.LookupEnv(key)
	if !ok {
		return nil, nil //nolint:nilnil
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		return nil, fmt.Errorf("invalid value %q for environment variable %q", v, key)
	}
	return &b, nil
}

// GetBool returns the boolean value of the environment variable.
// Returns false if the variable is not set or cannot be parsed as boolean.
// It does not return errors since it assumes that validation has already been done during startup.
func GetBool(key string) bool {
	// An unparsable value is reported as a configuration error during startup.
	b, _ := LookupBool(key)
	if b == nil {
		return false
	}
	return *b
}

// GetStringSlice returns the string slice value of the environment variable.
// Returns an empty slice if the variable is not set.
func GetStringSlice(key string) []string {
	v, ok := os.LookupEnv(key)
	if !ok || v == "" {
		return []string{}
	}

	return strings.Split(v, ",")
}
