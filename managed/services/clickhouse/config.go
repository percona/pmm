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

package clickhouse

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const (
	defaultClickhouseConfig = "default"
	clickhouseConfigDir     = "/etc/clickhouse-server"
)

// GetClickhouseConfig returns the config name if the matching
// <config>-config.xml and <config>-users.xml files exist on disk.
// Empty input falls back to defaultClickhouseConfig.
func GetClickhouseConfig(config string) (string, error) {
	if config == "" {
		return defaultClickhouseConfig, nil
	}

	return config, validateClickhouseConfigAt(config, clickhouseConfigDir)
}

// validateClickhouseConfigAt returns an error if configuration files are missing for given config
func validateClickhouseConfigAt(config, dir string) error {
	availableConfigs, err := availableClickhouseConfigs(dir)
	if err != nil {
		return fmt.Errorf("unable to get available ClickHouse configs: %w", err)
	}

	for _, suffix := range []string{"-config.xml", "-users.xml"} {
		path := filepath.Join(dir, config+suffix)
		if _, err := os.Stat(path); err != nil {
			if errors.Is(err, os.ErrNotExist) {
				return fmt.Errorf(
					"invalid PMM_CLICKHOUSE_CONFIG=%q: %s not found; available configs: %v",
					config, path, availableConfigs)
			}
			return fmt.Errorf("cannot stat %s: %w", path, err)
		}
	}
	return nil
}

// availableClickhouseConfigs lists config names that are present in the dir
func availableClickhouseConfigs(dir string) ([]string, error) {
	var configs []string

	matches, err := filepath.Glob(filepath.Join(dir, "*-config.xml"))
	if err != nil {
		return nil, err
	}
	for _, m := range matches {
		name := strings.TrimSuffix(filepath.Base(m), "-config.xml")
		configs = append(configs, name)
	}

	sort.Strings(configs)
	return configs, nil
}
