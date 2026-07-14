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

// Package collectors contains function to work with list of collectors.
package collectors

import (
	"fmt"
	"strings"
)

// FilterOutCollectors removes from exporter's flags disabled collectors.
// DisableCollector values should  match collector flag till end of string or till `=` sign.
// Examples:
// 1. If we pass `meminfo` then only "--collector.meminfo" but not "--collector.meminfo_numa".
// 2. If we pass `netstat.field` then "--collector.netstat.fields=^(.*_(InErrors|InErrs|InCsumErrors)..." should be disabled.
// 3. To disable "--collect.custom_query.hr" with directory ""--collect.custom_query.lr.directory" user should pass both names.
func FilterOutCollectors(prefix string, args, disabledCollectors []string) []string {
	disabledCollectorsMap := make(map[string]struct{})
	for _, disabledCollector := range disabledCollectors {
		key := fmt.Sprintf("%s%s", prefix, disabledCollector)
		disabledCollectorsMap[key] = struct{}{}
	}
	enabledArgs := []string{}
	for _, arg := range args {
		flagName := strings.Split(arg, "=")[0]
		if _, ok := disabledCollectorsMap[flagName]; !ok {
			enabledArgs = append(enabledArgs, arg)
		}
	}
	return enabledArgs
}

// DisableDefaultEnabledCollectors returns CLI arguments to disable default enabled collectors based on input.
// DefaultCollectors and disabledCollectors should be collector names without prefix.
// Result will be returned with prefix.
func DisableDefaultEnabledCollectors(prefix string, defaultCollectors []string, disabledCollectors []string) []string {
	defaultCollectorsMap := make(map[string]struct{})
	for _, defaultCollector := range defaultCollectors {
		defaultCollectorsMap[defaultCollector] = struct{}{}
	}
	args := []string{}
	for _, collector := range disabledCollectors {
		if _, ok := defaultCollectorsMap[collector]; ok {
			args = append(args, fmt.Sprintf("%s%s", prefix, collector))
		}
	}
	return args
}
